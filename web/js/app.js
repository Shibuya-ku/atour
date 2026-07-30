const state = {
  events: [],
  view: "matches", // matches | placements
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
  return p;
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
    const n = data.total ?? (data.items || []).length;
    $("emptyState").hidden = n > 0;
    $("stats").textContent = `共 ${data.total} 条`;
    if (state.view === "matches") renderMatches(data.items || []);
    else renderPlacements(data.items || []);
  } catch (err) {
    $("loadError").hidden = false;
    $("loadError").textContent = `加载失败：${err.message}`;
  }
}

function bind() {
  for (const id of ["q", "eventId", "gender", "belt", "style", "hideBye"]) {
    $(id).addEventListener("input", () => refresh());
    $(id).addEventListener("change", () => refresh());
  }
  $("viewMatches").addEventListener("click", () => {
    state.view = "matches";
    $("viewMatches").classList.add("active");
    $("viewPlacements").classList.remove("active");
    $("hideBye").closest("label").hidden = false;
    refresh();
  });
  $("viewPlacements").addEventListener("click", () => {
    state.view = "placements";
    $("viewPlacements").classList.add("active");
    $("viewMatches").classList.remove("active");
    $("hideBye").closest("label").hidden = true;
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
