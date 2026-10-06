const test = require("node:test");
const assert = require("node:assert/strict");
const fs = require("node:fs");
const paths = ["nginx/nginx.conf", "charts/nginx-html-manager/templates/configmap.yaml"];
function policy(path) {
  const text = fs.readFileSync(path, "utf8");
  const match = text.match(/add_header Content-Security-Policy "([^"]+)" always;/);
  assert.ok(match, `${path}: CSP missing`);
  return new Map(match[1].split(";").map(part => part.trim()).filter(Boolean).map(part => {
    const [name, ...values] = part.split(/\s+/);
    return [name, values];
  }));
}
for (const path of paths) {
  test(`${path}: scripts, storage, downloads, dialogs and documentation`, () => {
    const csp = policy(path);
    for (const flag of ["allow-scripts", "allow-same-origin", "allow-downloads", "allow-modals", "allow-popups", "allow-popups-to-escape-sandbox"]) {
      assert.ok(csp.get("sandbox").includes(flag), flag);
    }
    assert.deepEqual(csp.get("script-src"), ["'unsafe-inline'"]);
    assert.deepEqual(csp.get("style-src"), ["'unsafe-inline'"]);
  });
  test(`${path}: Git APIs over HTTPS without unsafe-eval`, () => {
    const csp = policy(path);
    assert.deepEqual(csp.get("connect-src"), ["https:"]);
    assert.ok(!csp.get("script-src").includes("'unsafe-eval'"));
  });
  test(`${path}: forms, frames, plugins and external scripts stay blocked`, () => {
    const csp = policy(path);
    for (const name of ["default-src", "form-action", "frame-src", "object-src", "base-uri", "frame-ancestors"]) {
      assert.deepEqual(csp.get(name), ["'none'"]);
    }
    assert.ok(!csp.get("sandbox").includes("allow-forms"));
  });
}
test("Docker and Helm use identical public policies", () => {
  assert.deepEqual(policy(paths[0]), policy(paths[1]));
});
test("interactive policy is only attached to the public server", () => {
  for (const path of paths) {
    const text = fs.readFileSync(path, "utf8");
    const at = text.indexOf("listen 8081;");
    assert.ok(at >= 0);
    assert.ok(!text.slice(0, at).includes("add_header Content-Security-Policy"));
  }
});
