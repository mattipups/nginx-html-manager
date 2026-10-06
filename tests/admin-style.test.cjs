const test = require("node:test");
const assert = require("node:assert/strict");
const fs = require("node:fs");

const adminCSS = fs.readFileSync("cmd/server/web/style.css", "utf8");
const main = fs.readFileSync("cmd/server/main.go", "utf8");
const publicCSS = main.match(/<style>\s*([\s\S]*?)<\/style>/)?.[1];
assert.ok(publicCSS, "public overview stylesheet must exist");

function rules(css, dark = false) {
  const media = css.match(/@media\(prefers-color-scheme:dark\)\s*\{([\s\S]*?)\n\}/);
  assert.ok(media, "dark theme must exist");
  const scope = dark ? media[1] : css.slice(0, media.index);
  const result = new Map();
  for (const match of scope.matchAll(/([^{}]+)\{([^{}]*)\}/g)) {
    const properties = Object.fromEntries(match[2].split(";").filter(Boolean).map(declaration => {
      const separator = declaration.indexOf(":");
      return [declaration.slice(0, separator).trim(), declaration.slice(separator + 1).trim()];
    }));
    for (const selector of match[1].split(",")) {
      const key = selector.trim();
      result.set(key, {...result.get(key), ...properties});
    }
  }
  return result;
}

function same(adminSelector, publicSelector, properties, dark = false) {
  const admin = rules(adminCSS, dark).get(adminSelector);
  const frontend = rules(publicCSS, dark).get(publicSelector);
  assert.ok(admin, adminSelector); assert.ok(frontend, publicSelector);
  for (const property of properties) {
    assert.notEqual(frontend[property], undefined, `${publicSelector}: ${property} must be defined`);
    assert.equal(admin[property], frontend[property], `${adminSelector}: ${property} (${dark ? "dark" : "light"})`);
  }
}

test("admin typography matches the public overview", () => {
  same("body", "body", ["font", "background", "color"]);
  same("h1", "h1", ["font-size", "font-weight", "margin"]);
  same(".heading h2", ".heading h2", ["font-size", "margin"]);
  same(".eyebrow", ".eyebrow", ["font-size", "letter-spacing", "font-weight", "text-transform", "color"]);
});

test("admin table and surfaces match the public overview", () => {
  same("section", "section", ["background", "border", "border-radius", "box-shadow"]);
  same("th", "th", ["font-size", "color", "text-transform", "letter-spacing", "font-weight"]);
  same("td", "td", ["padding", "border-bottom"]);
  same("td:first-child", "td:first-child", ["font-weight"]);
  same("footer", "footer", ["font-size", "color", "text-align"]);
});

test("admin search and actions match the public overview", () => {
  same("input[type=search]", "input[type=search]", ["font", "padding", "border-radius", "background", "border"]);
  for (const selector of ["button", ".action"]) same(selector, "a.action", ["font-size", "font-weight", "padding", "border-radius", "background", "color"]);
  same("button:hover", "a.action:hover", ["background", "border-color"]);
});

test("admin links match all public link color states", () => {
  for (const dark of [false, true]) {
    same("a:not(.action)", "td:first-child a", ["color"], dark);
    same("a:not(.action):visited", "td:first-child a:visited", ["color"], dark);
    same("a:not(.action):hover", "td:first-child a:hover", ["color"], dark);
    same("a:not(.action):focus-visible", "td:first-child a:focus-visible", ["color"], dark);
  }
});

test("admin dark palette matches the public overview", () => {
  same("body", "body", ["background", "color"], true);
  same("section", "section", ["background", "border-color"], true);
  same("th", "th", ["color", "border-color"], true);
  same("input[type=search]", "input[type=search]", ["background", "color", "border-color"], true);
  same(".action", "a.action", ["background", "color", "border-color"], true);
  same("footer", "footer", ["color"], true);
});

test("matching profile badges remain unchanged", () => {
  same(".badge", ".badge", ["padding", "border-radius", "font-size", "font-weight"]);
  for (const dark of [false, true]) for (const selector of [".badge-static", ".badge-interactive", ".badge-api"]) same(selector, selector, ["background", "color"], dark);
});
