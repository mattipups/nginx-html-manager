const test = require("node:test");
const assert = require("node:assert/strict");
const fs = require("node:fs");
const vm = require("node:vm");

const main = fs.readFileSync("cmd/server/main.go", "utf8");
const adminCSS = fs.readFileSync("cmd/server/web/style.css", "utf8");
const publicCSS = main.match(/<style>\s*([\s\S]*?)<\/style>/)[1];
const helper = fs.readFileSync("cmd/server/public_profile.go", "utf8");
const app = fs.readFileSync("cmd/server/web/app.js", "utf8");

function rules(css, dark = false) {
  const media = css.match(/@media\(prefers-color-scheme:dark\)\s*\{([\s\S]*?)\n\}/);
  assert.ok(media);
  const scope = dark ? media[1] : css.slice(0, media.index);
  const result = new Map();
  for (const match of scope.matchAll(/([^{}]+)\{([^{}]*)\}/g)) {
    const properties = Object.fromEntries(match[2].split(";").filter(Boolean).map(value => {
      const i = value.indexOf(":");
      return [value.slice(0, i).trim(), value.slice(i + 1).trim()];
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
  assert.ok(admin); assert.ok(frontend);
  for (const key of properties) {
    assert.notEqual(admin[key], undefined);
    assert.equal(frontend[key], admin[key], `${publicSelector}: ${key}, dark=${dark}`);
  }
}

test("public action links match backend colors, size, font and spacing", () => {
  same(".action", "a.action", ["font", "font-size", "font-weight", "color", "background", "padding", "margin", "border", "border-radius"]);
  same(".action:hover", "a.action:hover", ["background", "border-color"]);
  same(".action", "a.action", ["background", "color", "border-color"], true);
  same(".action:hover", "a.action:hover", ["background"], true);
});

test("public filename links use backend colors and inherited typography", () => {
  same("body", "body", ["font"]);
  same("td:first-child", "td:first-child", ["font-weight"]);
  assert.equal(rules(publicCSS).get("td:first-child a").font, "inherit");
  assert.ok(!main.includes("<strong><a"));
  for (const dark of [false, true]) {
    for (const state of ["", ":visited", ":hover", ":focus-visible"]) {
      same(`a:not(.action)${state}`, `td:first-child a${state}`, ["color"], dark);
    }
  }
  same(":focus-visible", ":focus-visible", ["outline", "outline-offset"]);
});

test("public profile labels and classes match the backend mapping", () => {
  const context = vm.createContext({});
  for (const name of ["profileLabel", "profileBadgeClass"]) {
    const fn = app.match(new RegExp(`function ${name}\\(profile\\) \\{[\\s\\S]*?\\n\\}`))[0];
    vm.runInContext(fn, context);
  }
  for (const profile of ["static", "interactive-local", "interactive-api", ""]) {
    const match = helper.match(new RegExp(`case "${profile}":\\s*return "([^"]+)", "([^"]+)"`));
    assert.ok(match, `public mapping missing for ${profile}`);
    assert.equal(match[1], context.profileBadgeClass(profile));
    assert.equal(match[2], context.profileLabel(profile));
  }
  assert.ok(helper.includes('fmt.Sprintf("Unbekannt (%s)", profile)'));
  assert.ok(main.includes("publicProfileBadge(p.Profile)"));
  assert.ok(main.includes("html.EscapeString(badgeLabel)"));
});

test("profile badge appearance matches in light and dark themes", () => {
  same(".badge", ".badge", ["padding", "border-radius", "font-size", "font-weight"]);
  for (const dark of [false, true]) {
    for (const selector of [".badge-static", ".badge-interactive", ".badge-api"]) {
      same(selector, selector, ["background", "color"], dark);
    }
  }
});
