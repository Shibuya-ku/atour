import { test } from "node:test";
import assert from "node:assert/strict";
import { parseDivision, filterMatches, filterPlacements } from "./filter.js";

test("parseDivision extracts gender belt style", () => {
  assert.deepEqual(parseDivision("Men's GI / Blue / Amateur / 69KG"), {
    gender: "Men's",
    belt: "Blue",
    style: "GI",
  });
  assert.deepEqual(parseDivision("Women's NO-GI / Purple / Professional / 52KG"), {
    gender: "Women's",
    belt: "Purple",
    style: "NO-GI",
  });
  assert.deepEqual(parseDivision("Men's GI / White / Amateur / 77KG"), {
    gender: "Men's",
    belt: "White",
    style: "GI",
  });
});

const sampleMatches = [
  {
    event_id: 1489,
    division: "Men's GI / Blue / Amateur / 69KG",
    is_bye: false,
    left_name: "Alice",
    left_club: "ATOS",
    right_name: "Bob",
    right_club: "JUMP MONGOLIA",
  },
  {
    event_id: 1489,
    division: "Men's GI / Blue / Amateur / 69KG",
    is_bye: true,
    left_name: "Alice",
    left_club: "ATOS",
    right_name: "BYE",
    right_club: "",
  },
  {
    event_id: 1533,
    division: "Women's NO-GI / Black / Professional / 52KG",
    is_bye: false,
    left_name: "Carol",
    left_club: "Gracie Barra",
    right_name: "Dana",
    right_club: "ATOS",
  },
];

test("filterMatches by q and event and hideBye", () => {
  const out = filterMatches(sampleMatches, {
    q: "jump",
    eventId: 1489,
    gender: "",
    belt: "",
    style: "",
    hideBye: true,
  });
  assert.equal(out.length, 1);
  assert.equal(out[0].right_club, "JUMP MONGOLIA");
});

test("filterMatches by gender belt style", () => {
  const out = filterMatches(sampleMatches, {
    q: "",
    eventId: 0,
    gender: "Women's",
    belt: "Black",
    style: "NO-GI",
    hideBye: false,
  });
  assert.equal(out.length, 1);
  assert.equal(out[0].left_name, "Carol");
});

test("filterPlacements by club", () => {
  const rows = [
    { event_id: 1489, division: "Men's GI / Blue / Amateur / 69KG", name: "Narangerel Dorjderem", club_name: "JUMP MONGOLIA" },
    { event_id: 1489, division: "Men's GI / White / Amateur / 69KG", name: "X", club_name: "ATOS" },
  ];
  const out = filterPlacements(rows, {
    q: "dorj",
    eventId: 1489,
    gender: "Men's",
    belt: "Blue",
    style: "GI",
  });
  assert.equal(out.length, 1);
  assert.equal(out[0].name, "Narangerel Dorjderem");
});
