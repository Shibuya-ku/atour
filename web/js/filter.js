/** @typedef {{ q?: string, eventId?: number|string, gender?: string, belt?: string, style?: string, hideBye?: boolean }} Query */

/**
 * @param {string} division
 */
export function parseDivision(division) {
  const d = String(division || "");
  let gender = "";
  if (d.startsWith("Men's")) gender = "Men's";
  else if (d.startsWith("Women's")) gender = "Women's";

  let belt = "";
  const beltM = d.match(/\/\s*(White|Blue|Purple|Brown|Black)\s*\//i);
  if (beltM) {
    belt = beltM[1][0].toUpperCase() + beltM[1].slice(1).toLowerCase();
  }

  let style = "";
  if (/\bNO-GI\b/i.test(d)) style = "NO-GI";
  else if (/\bGI\b/i.test(d)) style = "GI";

  return { gender, belt, style };
}

/**
 * @param {Query} query
 * @param {string} division
 */
function divisionMatches(query, division) {
  const p = parseDivision(division);
  if (query.gender && p.gender !== query.gender) return false;
  if (query.belt && p.belt !== query.belt) return false;
  if (query.style && p.style !== query.style) return false;
  return true;
}

function normEventId(id) {
  if (id === "" || id === null || id === undefined || id === 0 || id === "0") return 0;
  return Number(id);
}

/**
 * @param {Array<Record<string, any>>} matches
 * @param {Query} query
 */
export function filterMatches(matches, query) {
  const q = String(query.q || "").trim().toLowerCase();
  const eventId = normEventId(query.eventId);
  return matches.filter((m) => {
    if (eventId && m.event_id !== eventId) return false;
    if (query.hideBye && m.is_bye) return false;
    if (!divisionMatches(query, m.division)) return false;
    if (!q) return true;
    const blob = [m.left_name, m.left_club, m.right_name, m.right_club]
      .join(" ")
      .toLowerCase();
    return blob.includes(q);
  });
}

/**
 * @param {Array<Record<string, any>>} placements
 * @param {Query} query
 */
export function filterPlacements(placements, query) {
  const q = String(query.q || "").trim().toLowerCase();
  const eventId = normEventId(query.eventId);
  return placements.filter((p) => {
    if (eventId && p.event_id !== eventId) return false;
    if (!divisionMatches(query, p.division)) return false;
    if (!q) return true;
    const blob = [p.name, p.club_name, p.affiliation_name].join(" ").toLowerCase();
    return blob.includes(q);
  });
}
