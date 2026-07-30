const PAGE_SIZES = new Set([10, 20, 50, 100]);

const state = {
  events: [],
  view: "matches", // matches | placements | athletes
  page: 1,
  pageSize: 20,
  total: 0,
  selectedUserIds: new Set(),
  athleteSearchItems: [],
};

const $ = (id) => document.getElementById(id);

const FILTER_IDS = ["eventId", "gender", "belt", "style", "hideBye"];

function debounce(fn, ms) {
  let t;
  return (...args) => {
    clearTimeout(t);
    t = setTimeout(() => fn(...args), ms);
  };
}

function readQuery() {
  return {
    q: $("q").value,
    eventId: $("eventId").value,
    gender: $("gender").value,
    belt: $("belt").value,
    style: $("style").value,
    hideBye: $("hideBye").checked,
  };
}

function queryParams() {
  const q = readQuery();
  const p = new URLSearchParams();
  if (q.q) p.set("q", q.q);
  if (q.eventId && q.eventId !== "0") p.set("event_id", String(q.eventId));
  if (q.gender) p.set("gender", q.gender);
  if (q.belt) p.set("belt", q.belt);
  if (q.style) p.set("style", q.style);
  if (state.view === "matches" && q.hideBye) p.set("hide_bye", "1");
  p.set("limit", String(state.pageSize));
  p.set("offset", String((state.page - 1) * state.pageSize));
  return p;
}

function pageCount() {
  if (state.total <= 0) return 0;
  return Math.ceil(state.total / state.pageSize);
}

/** 当前页附近最多 5 个连续页码 */
function visiblePages(current, pages, max = 5) {
  if (pages <= 0) return [];
  if (pages <= max) {
    return Array.from({ length: pages }, (_, i) => i + 1);
  }
  let start = Math.max(1, current - Math.floor(max / 2));
  let end = start + max - 1;
  if (end > pages) {
    end = pages;
    start = end - max + 1;
  }
  return Array.from({ length: end - start + 1 }, (_, i) => start + i);
}

function updatePager() {
  const pages = pageCount();
  const pager = $("pager");
  pager.hidden = state.view === "athletes" || state.total <= 0;
  if (state.view === "athletes" || state.total <= 0) {
    if (state.view !== "athletes") $("stats").textContent = "共 0 条";
    $("pageNums").innerHTML = "";
    return;
  }
  const from = (state.page - 1) * state.pageSize + 1;
  const to = Math.min(state.page * state.pageSize, state.total);
  $("stats").textContent = `共 ${state.total} 条`;
  $("pageInfo").textContent = `第 ${state.page} / ${pages} 页 · 显示 ${from}–${to}`;
  $("pagePrev").disabled = state.page <= 1;
  $("pageNext").disabled = state.page >= pages;
  $("pageFirst").disabled = state.page <= 1;
  $("pageLast").disabled = state.page >= pages;

  const nums = $("pageNums");
  nums.innerHTML = "";
  for (const n of visiblePages(state.page, pages, 5)) {
    const btn = document.createElement("button");
    btn.type = "button";
    btn.textContent = String(n);
    btn.setAttribute("aria-label", `第 ${n} 页`);
    if (n === state.page) {
      btn.classList.add("active");
      btn.setAttribute("aria-current", "page");
    } else {
      btn.addEventListener("click", () => {
        state.page = n;
        refresh();
      });
    }
    nums.appendChild(btn);
  }
}

function resetPage() {
  state.page = 1;
}

function eventTitle(id) {
  const ev = state.events.find((e) => e.event_id === id);
  if (!ev) return String(id);
  const short = ev.title.split(" | ")[0];
  return `${ev.date_text || ""} · ${short}`.trim();
}

function fillEvents() {
  const sel = $("eventId");
  const keep = sel.value || "0";
  sel.innerHTML = `<option value="0">全部赛事</option>`;
  for (const e of state.events) {
    if (e.brackets_unavailable) continue;
    const opt = document.createElement("option");
    opt.value = String(e.event_id);
    opt.textContent = `${e.date_text || e.event_id} — ${e.location.split(",")[0]}`;
    sel.appendChild(opt);
  }
  sel.value = keep;
}

function opponentBadge(n) {
  const count = Number(n) || 0;
  return `<span class="opp-badge" title="该组别报名人数减 1">对手 ${count}</span>`;
}

function renderMatches(rows) {
  $("resultHead").innerHTML = `<tr>
    <th>赛事</th><th>组别</th><th>轮次</th><th>选手</th><th>胜负</th><th>比分</th>
  </tr>`;
  const tb = $("resultBody");
  tb.innerHTML = "";
  for (const m of rows) {
    const tr = document.createElement("tr");
    if (m.winner_side === "left") tr.className = "win-left";
    if (m.winner_side === "right") tr.className = "win-right";
    const score = [m.score_text, m.penalty_text ? `P ${m.penalty_text}` : ""]
      .filter(Boolean)
      .join(" · ");
    tr.innerHTML = `
      <td>${eventTitle(m.event_id)}</td>
      <td><div>${escapeHtml(m.division)}</div>${opponentBadge(m.opponent_count)}</td>
      <td>${escapeHtml(m.round_name || "")}<div class="muted">${escapeHtml(m.mat_match_nr || "")}</div></td>
      <td>
        <div class="p-left">${escapeHtml(m.left_name)} <span class="muted">${escapeHtml(m.left_club || "")}</span></div>
        <div class="p-right">${escapeHtml(m.right_name)} <span class="muted">${escapeHtml(m.right_club || "")}</span></div>
      </td>
      <td>${escapeHtml(m.won_by || "")}</td>
      <td>${escapeHtml(score)}</td>`;
    tb.appendChild(tr);
  }
}

function renderPlacements(rows) {
  $("resultHead").innerHTML = `<tr>
    <th>赛事</th><th>组别</th><th>名次</th><th>选手</th><th>俱乐部</th><th>联盟</th>
  </tr>`;
  const tb = $("resultBody");
  tb.innerHTML = "";
  const sorted = [...rows].sort((a, b) => {
    if (a.event_id !== b.event_id) return a.event_id - b.event_id;
    if (a.division !== b.division) return a.division.localeCompare(b.division);
    return a.placement - b.placement;
  });
  for (const p of sorted) {
    const tr = document.createElement("tr");
    tr.innerHTML = `
      <td>${eventTitle(p.event_id)}</td>
      <td><div>${escapeHtml(p.division)}</div>${opponentBadge(p.opponent_count)}</td>
      <td>${p.placement}</td>
      <td>${escapeHtml(p.name)}</td>
      <td>${escapeHtml(p.club_name || "")}</td>
      <td class="muted">${escapeHtml(p.affiliation_name || "")}</td>`;
    tb.appendChild(tr);
  }
}

function escapeHtml(s) {
  return String(s)
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;");
}

function formatPlacement(entry) {
  if (entry.placement_label) return entry.placement_label;
  if (entry.placement === 0 || entry.placement === "0") return "无正式名次";
  if (entry.placement != null) return String(entry.placement);
  return "—";
}

function formatMapCounts(obj) {
  if (!obj || typeof obj !== "object") return "";
  return Object.entries(obj)
    .sort((a, b) => b[1] - a[1])
    .map(([k, v]) => `${k} ${v}`)
    .join(" · ");
}

function updateAthleteLoadBtn() {
  $("athleteLoadProfile").disabled = state.selectedUserIds.size === 0;
}

function clearAthleteProfile() {
  $("athleteSummary").hidden = true;
  $("athleteTimelineWrap").hidden = true;
  $("athleteEncountersWrap").hidden = true;
  $("athleteTruncated").hidden = true;
  $("athleteSummary").innerHTML = "";
  $("athleteTimelineBody").innerHTML = "";
  $("athleteEncountersBody").innerHTML = "";
}

function renderAthleteIdentities(items) {
  const wrap = $("athleteIdentities");
  wrap.innerHTML = "";
  if (!items.length) {
    $("emptyState").hidden = false;
    $("emptyState").textContent = "无匹配选手身份";
    return;
  }
  $("emptyState").hidden = true;
  for (const item of items) {
    const card = document.createElement("label");
    card.className = "athlete-identity";
    if (item.user_id === 0) card.classList.add("warn");
    if (state.selectedUserIds.has(item.user_id)) card.classList.add("selected");

    const cb = document.createElement("input");
    cb.type = "checkbox";
    cb.value = String(item.user_id);
    cb.checked = state.selectedUserIds.has(item.user_id);
    cb.addEventListener("change", () => {
      if (cb.checked) state.selectedUserIds.add(item.user_id);
      else state.selectedUserIds.delete(item.user_id);
      card.classList.toggle("selected", cb.checked);
      updateAthleteLoadBtn();
      clearAthleteProfile();
    });

    const body = document.createElement("div");
    body.className = "athlete-identity-body";
    const warn = item.user_id === 0 ? '<span class="athlete-identity-warn">无账号 ID</span>' : "";
    const clubs = (item.clubs || []).join(" · ") || "—";
    body.innerHTML = `
      <div class="athlete-identity-name">${escapeHtml(item.name)}${warn}</div>
      <div class="athlete-identity-meta">
        ID ${item.user_id} · ${item.event_count} 站 · ${item.match_count} 场<br />
        最近：${escapeHtml(item.last_date_text || "—")}<br />
        ${escapeHtml(clubs)}
      </div>`;

    card.appendChild(cb);
    card.appendChild(body);
    wrap.appendChild(card);
  }
}

async function searchAthletes() {
  const q = $("q").value.trim();
  $("athleteSearchHint").hidden = true;
  clearAthleteProfile();
  if (q.length < 2) {
    state.athleteSearchItems = [];
    $("athleteIdentities").innerHTML = "";
    $("emptyState").hidden = true;
    if (q.length > 0) {
      $("athleteSearchHint").hidden = false;
    }
    return;
  }
  try {
    $("loadError").hidden = true;
    const res = await fetch(`/api/athletes/search?q=${encodeURIComponent(q)}`);
    if (!res.ok) throw new Error(`search ${res.status}`);
    const data = await res.json();
    state.athleteSearchItems = data.items || [];
    renderAthleteIdentities(state.athleteSearchItems);
    $("stats").textContent = `找到 ${state.athleteSearchItems.length} 个身份`;
  } catch (err) {
    $("loadError").hidden = false;
    $("loadError").textContent = `搜索失败：${err.message}`;
  }
}

const debouncedSearchAthletes = debounce(searchAthletes, 300);

function renderAthleteSummary(summary) {
  const el = $("athleteSummary");
  const belts = formatMapCounts(summary.belts);
  const styles = formatMapCounts(summary.styles);
  const clubs = (summary.clubs || [])
    .map((c) => `${c.name} (${c.count})`)
    .join(" · ");
  el.innerHTML = `
    <div><dt>组别</dt><dd>${summary.divisions}</dd></div>
    <div><dt>对阵</dt><dd>${summary.matches}</dd></div>
    <div><dt>胜 / 负</dt><dd>${summary.wins} / ${summary.losses}</dd></div>
    <div><dt>轮空</dt><dd>${summary.byes}</dd></div>
    <div><dt>金 / 银 / 铜</dt><dd>${summary.gold} / ${summary.silver} / ${summary.bronze}</dd></div>
    <div><dt>无正式名次</dt><dd>${summary.no_placement}</dd></div>
    <div class="stat-group">
      ${belts ? `<span>腰带<strong>${escapeHtml(belts)}</strong></span>` : ""}
      ${styles ? `<span>赛制<strong>${escapeHtml(styles)}</strong></span>` : ""}
      ${clubs ? `<span>俱乐部<strong>${escapeHtml(clubs)}</strong></span>` : ""}
    </div>`;
  el.hidden = false;
}

function renderAthleteTimeline(timeline) {
  $("athleteTimelineHead").innerHTML = `<tr>
    <th>赛事</th><th>组别</th><th>俱乐部</th><th>名次</th><th>胜/负/轮空</th>
  </tr>`;
  const tb = $("athleteTimelineBody");
  tb.innerHTML = "";
  for (const row of timeline) {
    const tr = document.createElement("tr");
    const title = row.title || eventTitle(row.event_id);
    const dateLoc = [row.date_text, row.location].filter(Boolean).join(" · ");
    tr.innerHTML = `
      <td><div>${escapeHtml(title)}</div><div class="muted">${escapeHtml(dateLoc)}</div></td>
      <td><div>${escapeHtml(row.division)}</div>${opponentBadge(row.opponent_count)}</td>
      <td>${escapeHtml(row.club || "")}</td>
      <td>${escapeHtml(formatPlacement(row))}</td>
      <td>${row.wins} / ${row.losses} / ${row.byes}</td>`;
    tb.appendChild(tr);
  }
  $("athleteTimelineWrap").hidden = timeline.length === 0;
}

function encounterResultClass(result) {
  if (result === "win") return "enc-result-win";
  if (result === "loss") return "enc-result-loss";
  if (result === "bye") return "enc-result-bye";
  return "enc-result-unknown";
}

function encounterResultLabel(result) {
  if (result === "win") return "胜";
  if (result === "loss") return "负";
  if (result === "bye") return "轮空";
  return "—";
}

function renderAthleteEncounters(encounters) {
  $("athleteEncountersHead").innerHTML = `<tr>
    <th>赛事</th><th>组别</th><th>轮次</th><th>对手</th><th>结果</th><th>方式</th><th>比分</th>
  </tr>`;
  const tb = $("athleteEncountersBody");
  tb.innerHTML = "";
  for (const enc of encounters) {
    const tr = document.createElement("tr");
    const title = enc.title || eventTitle(enc.event_id);
    tr.innerHTML = `
      <td><div>${escapeHtml(title)}</div><div class="muted">${escapeHtml(enc.date_text || "")}</div></td>
      <td>${escapeHtml(enc.division)}</td>
      <td>${escapeHtml(enc.round_name || "")}</td>
      <td>${escapeHtml(enc.opponent_name)} <span class="muted">${escapeHtml(enc.opponent_club || "")}</span></td>
      <td class="${encounterResultClass(enc.result)}">${encounterResultLabel(enc.result)}</td>
      <td>${escapeHtml(enc.won_by || "")}</td>
      <td>${escapeHtml(enc.score_text || "")}</td>`;
    tb.appendChild(tr);
  }
  $("athleteEncountersWrap").hidden = encounters.length === 0;
}

async function loadAthleteProfile() {
  if (state.selectedUserIds.size === 0) return;
  const ids = [...state.selectedUserIds].join(",");
  try {
    $("loadError").hidden = true;
    const res = await fetch(`/api/athletes/profile?user_ids=${ids}`);
    if (!res.ok) throw new Error(`profile ${res.status}`);
    const data = await res.json();
    renderAthleteSummary(data.summary || {});
    renderAthleteTimeline(data.timeline || []);
    renderAthleteEncounters(data.encounters || []);
    $("athleteTruncated").hidden = !data.truncated;
    const tl = (data.timeline || []).length;
    const enc = (data.encounters || []).length;
    $("stats").textContent = `档案：${tl} 条时间线 · ${enc} 场对阵`;
  } catch (err) {
    $("loadError").hidden = false;
    $("loadError").textContent = `档案加载失败：${err.message}`;
  }
}

function setMatchFiltersVisible(visible) {
  for (const id of FILTER_IDS) {
    const el = $(id);
    const label = el.closest("label");
    if (label) label.hidden = !visible;
  }
  if (visible) {
    $("hideBye").closest("label").hidden = state.view === "placements";
  }
}

function setViewMode(view) {
  state.view = view;
  $("viewMatches").classList.toggle("active", view === "matches");
  $("viewPlacements").classList.toggle("active", view === "placements");
  $("viewAthletes").classList.toggle("active", view === "athletes");

  const isAthletes = view === "athletes";
  setMatchFiltersVisible(!isAthletes);
  $("q").placeholder = isAthletes ? "选手姓名" : "选手 / 俱乐部";
  $("resultTableWrap").hidden = isAthletes;
  $("pager").hidden = isAthletes || state.total <= 0;
  $("athletePanel").hidden = !isAthletes;
  $("emptyState").hidden = true;
  $("athleteSearchHint").hidden = true;

  if (isAthletes) {
    clearAthleteProfile();
    const q = $("q").value.trim();
    if (q.length >= 2) searchAthletes();
    else $("stats").textContent = "";
  } else {
    state.selectedUserIds.clear();
    updateAthleteLoadBtn();
    $("athleteIdentities").innerHTML = "";
    resetPage();
    refresh();
  }
}

async function loadEvents() {
  const res = await fetch("/api/events");
  if (!res.ok) throw new Error(`events ${res.status}`);
  const data = await res.json();
  state.events = data.items || [];
  fillEvents();
}

async function refresh() {
  if (state.view === "athletes") return;
  try {
    $("loadError").hidden = true;
    const p = queryParams();
    const path = state.view === "matches" ? "/api/matches" : "/api/placements";
    const res = await fetch(`${path}?${p}`);
    if (!res.ok) throw new Error(`${path} ${res.status}`);
    const data = await res.json();
    state.total = Number(data.total) || 0;
    const pages = pageCount();
    if (pages > 0 && state.page > pages) {
      state.page = pages;
      await refresh();
      return;
    }
    $("emptyState").hidden = state.total > 0;
    $("emptyState").textContent = "无匹配结果";
    updatePager();
    if (state.view === "matches") renderMatches(data.items || []);
    else renderPlacements(data.items || []);
  } catch (err) {
    $("loadError").hidden = false;
    $("loadError").textContent = `加载失败：${err.message}`;
  }
}

function onFilterChange() {
  if (state.view === "athletes") {
    debouncedSearchAthletes();
    return;
  }
  resetPage();
  refresh();
}

function bind() {
  for (const id of ["q", "eventId", "gender", "belt", "style", "hideBye"]) {
    $(id).addEventListener("input", onFilterChange);
    $(id).addEventListener("change", onFilterChange);
  }
  $("pageSize").addEventListener("change", () => {
    const n = Number($("pageSize").value);
    state.pageSize = PAGE_SIZES.has(n) ? n : 20;
    resetPage();
    refresh();
  });
  $("pageFirst").addEventListener("click", () => {
    if (state.page <= 1) return;
    state.page = 1;
    refresh();
  });
  $("pagePrev").addEventListener("click", () => {
    if (state.page <= 1) return;
    state.page -= 1;
    refresh();
  });
  $("pageNext").addEventListener("click", () => {
    if (state.page >= pageCount()) return;
    state.page += 1;
    refresh();
  });
  $("pageLast").addEventListener("click", () => {
    const pages = pageCount();
    if (state.page >= pages) return;
    state.page = pages;
    refresh();
  });
  $("viewMatches").addEventListener("click", () => setViewMode("matches"));
  $("viewPlacements").addEventListener("click", () => setViewMode("placements"));
  $("viewAthletes").addEventListener("click", () => setViewMode("athletes"));
  $("athleteLoadProfile").addEventListener("click", loadAthleteProfile);
}

async function load() {
  try {
    $("loadError").hidden = true;
    await loadEvents();
    await refresh();
  } catch (err) {
    $("loadError").hidden = false;
    $("loadError").textContent =
      `加载失败：${err.message}。请先 go run ./cmd/ajpdb import -from output -dsn data/atour.db，再 go run ./cmd/ajpweb -dsn data/atour.db`;
  }
}

bind();
load();
