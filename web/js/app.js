const PAGE_SIZES = new Set([10, 20, 50, 100]);

const state = {
  events: [],
  view: "matches", // matches | placements
  page: 1,
  pageSize: 20,
  total: 0,
};

const $ = (id) => document.getElementById(id);

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
  pager.hidden = state.total <= 0;
  if (state.total <= 0) {
    $("stats").textContent = "共 0 条";
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

async function loadEvents() {
  const res = await fetch("/api/events");
  if (!res.ok) throw new Error(`events ${res.status}`);
  const data = await res.json();
  state.events = data.items || [];
  fillEvents();
}

async function refresh() {
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
    updatePager();
    if (state.view === "matches") renderMatches(data.items || []);
    else renderPlacements(data.items || []);
  } catch (err) {
    $("loadError").hidden = false;
    $("loadError").textContent = `加载失败：${err.message}`;
  }
}

function onFilterChange() {
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
  $("viewMatches").addEventListener("click", () => {
    state.view = "matches";
    $("viewMatches").classList.add("active");
    $("viewPlacements").classList.remove("active");
    $("hideBye").closest("label").hidden = false;
    resetPage();
    refresh();
  });
  $("viewPlacements").addEventListener("click", () => {
    state.view = "placements";
    $("viewPlacements").classList.add("active");
    $("viewMatches").classList.remove("active");
    $("hideBye").closest("label").hidden = true;
    resetPage();
    refresh();
  });
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
