const test = require("node:test");
const assert = require("node:assert/strict");
const fs = require("node:fs");
const read = path => fs.readFileSync(path,"utf8");

test("release version and both UIs use 1.5.0",()=>{
  assert.equal(read("VERSION").trim(),"1.5.0");
  const admin=read("cmd/server/web/index.html"), frontend=read("cmd/server/main.go");
  assert.match(admin,/<title>HTML-Verwaltung 1\.5\.0<\/title>/);
  assert.match(admin,/HTML-PUBLISHER · 1\.5\.0/);
  assert.match(frontend,/HTML-PUBLISHER · 1\.5\.0/);
});
test("Compose, Helm and scanner image tags are synchronized",()=>{
  const compose=read("compose.yaml"), values=read("charts/nginx-html-manager/values.yaml"), chart=read("charts/nginx-html-manager/Chart.yaml"), scan=read("scripts/scan.sh");
  for(const image of ["local/html-manager","local/html-manager-nginx"]) {
    assert.ok(compose.includes(`image: ${image}:1.5.0`));
    assert.ok(values.includes(`repository: ${image}, tag: "1.5.0"`));
    assert.ok(scan.includes(`image=${image}:1.5.0`));
  }
  assert.match(chart,/^version: 1\.5\.0$/m);assert.match(chart,/^appVersion: "1\.5\.0"$/m);
  assert.match(compose,/test: \[CMD, \/app\/server, healthcheck\]/);
});
test("public logo is rendered before the heading with matching responsive dimensions",()=>{
  const frontend=read("cmd/server/main.go"), helper=read("cmd/server/public_logo.go");
  assert.ok(frontend.indexOf("b.WriteString(logoPicture)")<frontend.indexOf('b.WriteString(`<p class="eyebrow">'));
  assert.ok(frontend.includes(".brand{display:block;max-width:360px}"));
  assert.ok(frontend.includes(".brand img{display:block;width:100%;height:auto}"));
  assert.ok(helper.includes('media="(prefers-color-scheme: dark)"'));
  assert.ok(helper.includes('width="1400" height="700"'));
  assert.ok(helper.includes("data:image/png;base64,"));
});
