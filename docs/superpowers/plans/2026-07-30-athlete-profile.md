# 选手个人档案 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在现有查询页增加「选手」视图：姓名搜索 → 人工勾选 `user_id` 身份 → 展示时间线、汇总指标与对手对阵明细（不做交手次数聚合）。

**Architecture:** 在 `internal/store` 增加 `SearchAthletes` / `AthleteProfile`（基于现有 matches/placements/events 聚合）；`cmd/ajpweb` 暴露 `GET /api/athletes/search` 与 `GET /api/athletes/profile`；前端在同页切换选手模式，勾选身份后拉档案。不持久化合并、不改爬虫。

**Tech Stack:** Go 1.25、`database/sql`（SQLite/MySQL）、原生 HTML/CSS/JS

**Branch:** `feature/athlete-profile`（已从含 design spec 的 main tip 切出）

**Spec:** `docs/superpowers/specs/2026-07-30-athlete-profile-design.md`

## Global Constraints

- `user_id` 不能作为终身主键；合并仅当次勾选，不写合并表
- 不能按俱乐部自动合并身份
- 对手池不做「交手次数」汇总，只列当场对阵明细
- `q` 长度 `< 2` → search 返回 HTTP 400
- search 分组键为 `user_id`；`user_id = 0` 可成组但 UI 需警示
- search 最多 50 组；timeline ≤ 200、encounters ≤ 500；截断时 `truncated: true`
- `placement = 0` → 汇总 `no_placement`，UI 文案「无正式名次」
- BYE：计入 `byes`，不计入 `wins`/`losses`
- 最小 diff：不引入前端构建、不新增爬虫字段、不做俱乐部看板
- 提交仅在本分支；未经用户要求不 `git push`

---

## File Structure

| 路径 | 职责 |
|------|------|
| `internal/store/athlete.go` | 类型：`AthleteIdentity`、`AthleteSummary`、`TimelineEntry`、`Encounter`、`AthleteProfile`；实现 `SearchAthletes` / `BuildAthleteProfile`（或 `AthleteProfile` 方法） |
| `internal/store/store.go` | Store 接口增加两个方法 |
| `internal/store/athlete_test.go` | 搜索与档案聚合单测 |
| `cmd/ajpweb/api.go` | 两个 HTTP handler + 路由挂载 |
| `cmd/ajpweb/main.go` / `main_test.go` | 路由注册；API 测试 |
| `web/index.html` | 「选手」按钮、身份列表区、档案区 markup |
| `web/css/app.css` | 选手视图样式 |
| `web/js/app.js` | 选手模式状态机与渲染 |
| `README.md` | 补充选手档案能力一句 |

**复用：** `ParseDivision`（`internal/store/division.go`）、现有 `EventRow` / 表结构。

---

### Task 1: Store 类型与 SearchAthletes

**Files:**
- Modify: `internal/store/store.go`
- Create: `internal/store/athlete.go`
- Test: `internal/store/athlete_test.go`

**Interfaces:**
- Consumes: `SQLStore`、`ReplaceAll`、现有表
- Produces:
  - `type AthleteIdentity struct { UserID int; Name string; Clubs []string; EventCount, MatchCount int; LastDateText string }`
  - `func (s *SQLStore) SearchAthletes(ctx context.Context, q string, limit int) ([]AthleteIdentity, error)`
  - Store 接口增加 `SearchAthletes(ctx, q string, limit int) ([]AthleteIdentity, error)`
  - `limit <= 0` 时默认 50；硬顶 50

- [ ] **Step 1: Write the failing test**

```go
func TestSearchAthletesGroupsByUserID(t *testing.T) {
	ctx := context.Background()
	s, err := Open("sqlite", filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	_ = s.Migrate(ctx)
	_ = s.ReplaceAll(ctx,
		[]EventRow{{EventID: 1, Title: "E", DateText: "1 Jan 2025", Location: "China"}},
		[]ajp.MatchRecord{
			{EventID: 1, BracketID: 10, Division: "Men's / Blue / GI", MatchID: 1,
				LeftName: "Zhiyuan Kong", LeftClub: "ClubA", LeftUserID: 100,
				RightName: "Other", RightUserID: 200, WinnerSide: "left"},
			{EventID: 1, BracketID: 11, Division: "Men's / Blue / NO-GI", MatchID: 2,
				LeftName: "Zhiyuan Kong", LeftClub: "ClubB", LeftUserID: 100,
				RightName: "X", RightUserID: 201, WinnerSide: "right"},
		},
		[]ajp.PlacementRecord{
			{EventID: 1, BracketID: 10, Division: "Men's / Blue / GI", Placement: 1,
				UserID: 100, Name: "Zhiyuan Kong", ClubName: "ClubA"},
		},
	)
	items, err := s.SearchAthletes(ctx, "kong", 50)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].UserID != 100 {
		t.Fatalf("%+v", items)
	}
	if items[0].MatchCount < 1 || items[0].EventCount < 1 {
		t.Fatalf("%+v", items[0])
	}
	if len(items[0].Clubs) < 1 {
		t.Fatalf("clubs=%v", items[0].Clubs)
	}
}

func TestSearchAthletesRejectsShortQuery(t *testing.T) {
	// SearchAthletes 本身可返回 error 或空；HTTP 层做 400。
	// store 层：q trim 后 len < 2 返回 error ErrQueryTooShort
	ctx := context.Background()
	s, _ := Open("sqlite", filepath.Join(t.TempDir(), "t.db"))
	defer s.Close()
	_ = s.Migrate(ctx)
	_, err := s.SearchAthletes(ctx, "a", 50)
	if err == nil {
		t.Fatal("expected error")
	}
}
```

- [ ] **Step 2: Run test — expect FAIL**

Run: `go test ./internal/store/ -run TestSearchAthletes -v`

- [ ] **Step 3: Implement**

在 `athlete.go`：

```go
var ErrQueryTooShort = errors.New("query too short")

type AthleteIdentity struct {
	UserID       int
	Name         string
	Clubs        []string
	EventCount   int
	MatchCount   int
	LastDateText string
}
```

`SearchAthletes` 逻辑（可移植 SQL）：

1. 若 `len(strings.TrimSpace(q)) < 2` → `ErrQueryTooShort`
2. 从 matches 收集：`LEFT JOIN events`，`LOWER(left_name) LIKE %q%` → 记 `(left_user_id, left_name, left_club, event_id, date_text)`；右侧同理
3. 从 placements 收集：`LOWER(name) LIKE %q%` → `(user_id, name, club_name, event_id, date_text)`
4. 在 Go 中按 `user_id` 聚合：name 取众数；clubs 去重；event_id 集合大小；match 侧出现次数作 MatchCount；LastDateText 取「最近」——因 `date_text` 非 ISO，v1 用 **最大 event_id** 对应的 `date_text`（并在注释写明）
5. 按 `MatchCount` 降序，截断至 `limit`（默认/上限 50）

把方法挂到 `Store` 接口与 `*SQLStore`。

- [ ] **Step 4: Run tests — PASS**

Run: `go test ./internal/store/ -run TestSearchAthletes -v`

- [ ] **Step 5: Commit**

```bash
git add internal/store/
git commit -m "feat(store): SearchAthletes 按 user_id 分组搜索"
```

---

### Task 2: AthleteProfile 聚合（时间线 / 汇总 / 对手明细）

**Files:**
- Modify: `internal/store/athlete.go`
- Modify: `internal/store/store.go`
- Modify: `internal/store/athlete_test.go`

**Interfaces:**
- Consumes: `ParseDivision`、`SearchAthletes` 同库数据
- Produces:
  - `type AthleteSummary struct { Divisions, Matches, Wins, Losses, Byes, Gold, Silver, Bronze, NoPlacement int; Belts, Styles map[string]int; Clubs []ClubCount }`
  - `type ClubCount struct { Name string; Count int }`
  - `type TimelineEntry struct { EventID, BracketID int; Title, DateText, Location, Division, Club string; Placement *int; PlacementLabel string; OpponentCount, Wins, Losses, Byes int }`
  - `type Encounter struct { EventID, BracketID, MatchID int; Title, DateText, Division, RoundName, OpponentName, OpponentClub, Result, WonBy, ScoreText string }`
  - `type AthleteProfileResult struct { Identities []AthleteIdentity; Summary AthleteSummary; Timeline []TimelineEntry; Encounters []Encounter; Truncated bool }`
  - `func (s *SQLStore) AthleteProfile(ctx context.Context, userIDs []int) (AthleteProfileResult, error)`
  - 空 `userIDs` → error `ErrNoUserIDs`

**胜负归属：**

```go
func sideResult(userIDs map[int]bool, m ajp.MatchRecord) (win, loss, bye bool) {
	if m.IsBye {
		if userIDs[m.LeftUserID] || userIDs[m.RightUserID] {
			return false, false, true
		}
		return
	}
	onLeft := userIDs[m.LeftUserID]
	onRight := userIDs[m.RightUserID]
	if !onLeft && !onRight {
		return
	}
	switch m.WinnerSide {
	case "left":
		if onLeft { win = true } else { loss = true }
	case "right":
		if onRight { win = true } else { loss = true }
	}
	return
}
```

- [ ] **Step 1: Write failing tests**

```go
func TestAthleteProfileStatsAndEncounters(t *testing.T) {
	ctx := context.Background()
	s, _ := Open("sqlite", filepath.Join(t.TempDir(), "t.db"))
	defer s.Close()
	_ = s.Migrate(ctx)
	_ = s.ReplaceAll(ctx,
		[]EventRow{{EventID: 1, Title: "E1", DateText: "d1", Location: "China"}},
		[]ajp.MatchRecord{
			{EventID: 1, BracketID: 10, Division: "Men's / Purple / GI", MatchID: 1,
				LeftName: "A", LeftUserID: 1, LeftClub: "C1",
				RightName: "B", RightUserID: 2, RightClub: "C2",
				WinnerSide: "left", WonBy: "POINTS", ScoreText: "2-0", IsBye: false,
				OpponentCount: 3, RoundName: "Final"},
			{EventID: 1, BracketID: 10, Division: "Men's / Purple / GI", MatchID: 2,
				LeftName: "A", LeftUserID: 1, IsBye: true, OpponentCount: 3},
		},
		[]ajp.PlacementRecord{
			{EventID: 1, BracketID: 10, Division: "Men's / Purple / GI", Placement: 1, UserID: 1, Name: "A", ClubName: "C1"},
			{EventID: 1, BracketID: 10, Division: "Men's / Purple / GI", Placement: 0, UserID: 9, Name: "Z"}, // 无关
		},
	)
	prof, err := s.AthleteProfile(ctx, []int{1})
	if err != nil {
		t.Fatal(err)
	}
	if prof.Summary.Wins != 1 || prof.Summary.Byes != 1 || prof.Summary.Losses != 0 {
		t.Fatalf("%+v", prof.Summary)
	}
	if prof.Summary.Gold != 1 || prof.Summary.NoPlacement != 0 {
		t.Fatalf("%+v", prof.Summary)
	}
	if len(prof.Encounters) != 1 { // BYE 不进 encounters（或进？——规格：对手明细是对阵；BYE 无对手）
		t.Fatalf("encounters=%d", len(prof.Encounters))
	}
	if prof.Encounters[0].OpponentName != "B" || prof.Encounters[0].Result != "win" {
		t.Fatalf("%+v", prof.Encounters[0])
	}
	if len(prof.Timeline) != 1 || prof.Timeline[0].PlacementLabel == "无正式名次" {
		// placement 1 → 显示 "1" 或 label 非「无正式名次」
	}
}

func TestAthleteProfileNoPlacementAndMergeIDs(t *testing.T) {
	// user 1 与 2 勾选合并；placement 0 计入 no_placement
	// ...
}

func TestAthleteProfileEmptyUserIDs(t *testing.T) {
	s, _ := Open("sqlite", filepath.Join(t.TempDir(), "t.db"))
	defer s.Close()
	_, err := s.AthleteProfile(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error")
	}
}
```

**Encounters 规则（锁定）：** 仅非 BYE 且本人在一侧的比赛；每场一行；`Result` 为 `win`|`loss`|`unknown`。

**Timeline PlacementLabel：** `placement==0` → `"无正式名次"`；有名次 → `strconv.Itoa(p)`；无 placement 行 → `""` 且 `Placement` 指针 nil。

- [ ] **Step 2: Run — expect FAIL**

- [ ] **Step 3: Implement AthleteProfile**

1. 查所有 `left_user_id IN S OR right_user_id IN S` 的 matches（JOIN events）
2. 查所有 `user_id IN S` 的 placements（JOIN events）
3. 构建 timeline：按 `(event_id, bracket_id)` 聚合；排序：`event_id` 降序（date_text 不可靠）
4. 汇总 belts/styles：对 timeline 每条 `ParseDivision(division)`
5. clubs：从本人侧 club / placement.club_name 计数
6. 截断 timeline 200、encounters 500，设 `Truncated`
7. Identities：可对每个 user_id 再查众数 name/clubs，或从已加载行推导

- [ ] **Step 4: Run full store tests**

Run: `go test ./internal/store/ -v`

- [ ] **Step 5: Commit**

```bash
git add internal/store/
git commit -m "feat(store): AthleteProfile 时间线汇总与对手明细"
```

---

### Task 3: HTTP API

**Files:**
- Modify: `cmd/ajpweb/api.go`
- Modify: `cmd/ajpweb/main.go`（若路由在 newMux）
- Modify: `cmd/ajpweb/main_test.go`

**Interfaces:**
- Consumes: `store.SearchAthletes`、`store.AthleteProfile`
- Produces:
  - `GET /api/athletes/search?q=`
  - `GET /api/athletes/profile?user_ids=1,2,3`

- [ ] **Step 1: Write API tests**

```go
func TestAthleteSearchAndProfile(t *testing.T) {
	// 建临时 DB + ReplaceAll 最小数据
	// newMux(web, store)
	// GET /api/athletes/search?q=a → 400
	// GET /api/athletes/search?q=kong → 200 items
	// GET /api/athletes/profile?user_ids=100 → 200 summary
	// GET /api/athletes/profile → 400
}
```

- [ ] **Step 2: Run — expect FAIL**

- [ ] **Step 3: Implement handlers**

```go
func (s *apiServer) handleAthleteSearch(...) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	items, err := s.store.SearchAthletes(r.Context(), q, 50)
	if errors.Is(err, store.ErrQueryTooShort) {
		http.Error(w, "q too short", http.StatusBadRequest)
		return
	}
	// ...
	writeJSON(w, map[string]any{"items": items})
}

func (s *apiServer) handleAthleteProfile(...) {
	raw := r.URL.Query().Get("user_ids")
	// split by comma, Atoi, skip empty
	prof, err := s.store.AthleteProfile(ctx, ids)
	if errors.Is(err, store.ErrNoUserIDs) {
		http.Error(w, "user_ids required", 400)
		return
	}
	writeJSON(w, prof) // 确保 json tag 为 snake_case：在 store 类型上加 tag 或 DTO
}
```

在 `newMux` 注册：

```go
mux.HandleFunc("/api/athletes/search", api.handleAthleteSearch)
mux.HandleFunc("/api/athletes/profile", api.handleAthleteProfile)
```

**JSON：** 给 store 导出类型加 `json:"..."` snake_case，与 spec 一致。

- [ ] **Step 4: `go test ./cmd/ajpweb/ -v` PASS**

- [ ] **Step 5: Commit**

```bash
git add cmd/ajpweb/
git commit -m "feat(ajpweb): /api/athletes search 与 profile"
```

---

### Task 4: 前端选手视图

**Files:**
- Modify: `web/index.html`
- Modify: `web/css/app.css`
- Modify: `web/js/app.js`

**Interfaces:**
- Consumes: `/api/athletes/search`、`/api/athletes/profile`
- Produces: 选手模式 UI

- [ ] **Step 1: HTML 结构**

在 `view-toggle` 增加 `<button type="button" id="viewAthletes">选手</button>`。

增加区域（默认 hidden）：

```html
<section id="athletePanel" hidden>
  <div id="athleteIdentities" class="athlete-identities"></div>
  <button type="button" id="athleteLoadProfile" disabled>查看档案</button>
  <div id="athleteSummary" class="athlete-summary" hidden></div>
  <div id="athleteTimelineWrap" hidden>...</div>
  <div id="athleteEncountersWrap" hidden>...</div>
</section>
```

选手模式下隐藏：赛事/性别/腰带/赛制/BYE、对阵表、分页；关键词 placeholder 改为「选手姓名」。

- [ ] **Step 2: JS 状态与逻辑**

```javascript
// state.view: "matches" | "placements" | "athletes"
// state.selectedUserIds: Set<number>

async function searchAthletes() {
  const q = $("q").value.trim();
  if (q.length < 2) { /* 提示 */ return; }
  const res = await fetch(`/api/athletes/search?q=${encodeURIComponent(q)}`);
  // render checkboxes; user_id===0 加警示 class
}

async function loadAthleteProfile() {
  const ids = [...state.selectedUserIds].join(",");
  const res = await fetch(`/api/athletes/profile?user_ids=${ids}`);
  // render summary / timeline / encounters
  // placement_label 或 placement===0 →「无正式名次」
}
```

切换到对阵/名次时恢复原筛选 UI 并 `refresh()`。

选手模式：`q` 的 input 触发 search（可简单 debounce 300ms 或 change 时搜索）；「查看档案」在勾选变化后启用。

- [ ] **Step 3: CSS**

身份卡片网格、警示色、汇总条、与现有 dark theme 一致；避免新依赖。

- [ ] **Step 4: 手动验证**

```bash
go run ./cmd/ajpweb -dsn data/atour.db
```

搜索已知选手（如 Kong），勾选身份，确认三块数据与截断提示。

- [ ] **Step 5: Commit**

```bash
git add web/
git commit -m "feat(web): 选手视图搜索勾选与档案展示"
```

---

### Task 5: README 与收尾验证

**Files:**
- Modify: `README.md`

- [ ] **Step 1: README**

功能列表增加：选手档案（姓名搜索、人工勾选身份、时间线/汇总/对手明细）。  
查询页能力补充对应条目。

- [ ] **Step 2: 全量测试**

```bash
go test ./... -short
node --test web/js/filter.test.mjs
```

Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add README.md
git commit -m "docs: README 补充选手档案能力"
```

---

## Self-Review

1. **Spec coverage：** search/profile API、人工勾选、A/B/C 三块、截断、BYE/名次0、不持久化合并列于 Task 1–4。  
2. **无 TBD。** `date_text` 排序采用 `event_id` 已写明。  
3. **Encounters 不含 BYE** 已在 Task 2 锁定。  
4. **类型 json tag** 在 Task 3 要求补齐。

## 执行说明

实现时在分支 `feature/athlete-profile` 上工作。推荐 Subagent-Driven 或 Inline Execution。
