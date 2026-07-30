# 选手个人档案（Athlete Profile）Design

**日期：** 2026-07-30  
**状态：** 待实现  
**范围：** 在现有查询页内嵌选手视图；基于 SQLite/MySQL 已有 `matches` / `placements` / `events` 做聚合分析

## 背景与约束

atour 已具备中国站对阵/名次数据与分页查询。下一步优先做**选手个人档案**，而非俱乐部看板或宏观赛事分析。

现实约束（产品前提）：

- 选手可多次注册新账号 → **`user_id` 不能作为终身主键**
- 每次比赛俱乐部可能不同 → **不能按俱乐部自动合并身份**
- 对手也会改名/换号 → **对手池不做「交手次数」汇总**，只列当场对阵明细

## 目标（v1）

1. 按姓名搜索，列出疑似身份分组，由用户**人工勾选**后再聚合档案  
2. 档案包含：  
   - **A. 时间线**：参赛赛事/组别/名次/对阵摘要  
   - **B. 汇总**：胜负、奖牌、腰带与赛制分布、俱乐部履历列表  
   - **C. 对手明细**：一场一行，不做按对手聚合  
3. 合并仅作用于当次请求，**不持久化**人工合并关系  

## 非目标（v1）

- 持久化「同一人」映射表 / 记住合并  
- 对手交手次数、胜率排行榜  
- 俱乐部/地区对比看板、图表大屏  
- 新爬虫字段或改爬取范围  
- 鉴权、多用户账户体系  

## 方案选择

采用**方案 1：现有查询页内嵌「选手」视图**。

| 方案 | 说明 | 结论 |
|------|------|------|
| 1. 内嵌选手视图 | 复用 `ajpweb` + 现有 UI 风格 | **采用** |
| 2. 独立档案子页 | 解耦更强，首版成本高 | 不做 |
| 3. 仅统计 API/导出 | 无站内体验 | 不做 |

## 架构

```
[查询页] 对阵 | 名次 | 选手
                │
                ▼
     GET /api/athletes/search?q=
                │
     勾选 1..N 个 (user_id) 身份
                │
                ▼
     GET /api/athletes/profile?user_ids=1,2,3
                │
                ▼
     summary + timeline + encounters
```

- 存储：沿用 `internal/store`（SQLite 默认 / MySQL 可切换）  
- 新增查询方法：`SearchAthletes`、`AthleteProfile`  
- HTTP：扩展 `cmd/ajpweb/api.go`  
- 前端：`web/` 增加选手模式 UI，不引入构建工具  

## 身份模型

### 搜索分组

- 输入：姓名关键词 `q`（子串匹配 `matches` 两侧姓名与 `placements.name`）  
- `q` 长度 `< 2` → HTTP 400  
- 分组键：`user_id`（同一 `user_id` 下多名取出现次数最多者作为 `display_name`）  
- 每组附带：出现过的 `clubs[]`、`event_count`、`match_count`、`last_date_text`  
- `user_id = 0` 单独成组，UI 标注「无账号 ID」  
- 结果上限：最多 50 组  

### 档案聚合

- 入参：用户勾选的 `user_ids` 集合 `S`（至少一个）  
- 对阵归属：`left_user_id ∈ S` 或 `right_user_id ∈ S`  
- 名次归属：`user_id ∈ S`  
- 第一版不写合并表；刷新/清空勾选即回到未合并状态  

## 指标定义

### A. 时间线

按赛事日期（`events.date_text` / `event_id`）排序。  
每条 = 去重后的 `(event_id, bracket_id)` 出场：

- 赛事标题、日期、地点  
- `division`、当场俱乐部  
- 名次：有记录则显示；`placement = 0` 显示为「无正式名次」  
- `opponent_count`  
- 该组内本人相关对阵：胜 / 负 / 轮空条数  

### B. 汇总

| 字段 | 定义 |
|------|------|
| `divisions` | 时间线条目数 |
| `matches` | 非 BYE 的本人出场场次 |
| `wins` / `losses` | `winner_side` 指向本人 / 对方；无法判断不计 |
| `byes` | `is_bye` 且本人在场 |
| `gold` / `silver` / `bronze` | `placement ∈ {1,2,3}` |
| `no_placement` | `placement = 0` |
| `belts` | 按解析 belt 统计出场组别数 |
| `styles` | 按 GI / NO-GI 统计出场组别数 |
| `clubs` | 俱乐部去重列表（可附次数）；**不**用于身份合并 |

### C. 对手明细（encounters）

一行一场：赛事、组别、轮次、对手当场姓名与俱乐部、胜负、`won_by`、比分。  
**禁止**按对手姓名/`user_id` 做「交手 N 次」聚合。

## API 契约

### `GET /api/athletes/search?q=`

成功：

```json
{
  "items": [
    {
      "user_id": 409296,
      "name": "Zhiyuan Kong",
      "clubs": ["Simulation Jiujitsu", "Gracie Jiu-Jitsu Qingdao"],
      "event_count": 3,
      "match_count": 12,
      "last_date_text": "13-14 July 2025"
    }
  ]
}
```

### `GET /api/athletes/profile?user_ids=1,2,3`

- 非法/空 `user_ids` → 400  
- 无数据 → 200，空时间线与零汇总（不 404）  
- 硬上限：`timeline` 最多 200 条、`encounters` 最多 500 条；任一被截断则 `"truncated": true`

成功响应形状：

```json
{
  "identities": [{ "user_id": 1, "name": "...", "clubs": [] }],
  "summary": {
    "divisions": 0,
    "matches": 0,
    "wins": 0,
    "losses": 0,
    "byes": 0,
    "gold": 0,
    "silver": 0,
    "bronze": 0,
    "no_placement": 0,
    "belts": {},
    "styles": {},
    "clubs": []
  },
  "timeline": [],
  "encounters": [],
  "truncated": false
}
```

## UI 要点

- 顶部视图增加「选手」  
- 搜索 → 复选身份卡片 → 「查看档案」  
- 档案三块：汇总条 / 时间线表 / 对阵明细表  
- 文案：`placement = 0` →「无正式名次」；`user_id = 0` 警示  

## 测试要点

- 多 `user_id` 勾选后胜负与奖牌正确合并  
- BYE 计入 `byes`，不计入 `wins`/`losses`  
- `placement = 0` 进 `no_placement`  
- search 短查询 400；profile 空结果 200  
- encounters 行数 = 符合条件的对阵数，无按对手聚合  

## 后续可选（不在 v1）

- 持久化人工合并（本地 JSON / 表）  
- 对手侧同样的人工合并  
- 导出选手档案 CSV  
- 从对阵/名次行一键「打开选手」并预勾当前 `user_id`  
