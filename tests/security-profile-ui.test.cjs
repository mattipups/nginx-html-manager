const test = require("node:test");
const assert = require("node:assert/strict");
const fs = require("node:fs");
const vm = require("node:vm");

class Element {
  constructor(tag = "div") {
    this.tag = tag; this.children = []; this.value = ""; this.disabled = false;
    this.handlers = {}; this.clicks = 0; this.classList = {add() {}, remove() {}};
  }
  append(x) { this.children.push(x); }
  replaceChildren(...x) { this.children = x; }
  addEventListener(name, fn) { (this.handlers[name] ||= []).push(fn); }
  closest(selector) { return selector.split(", ").includes(this.tag) ? this : null; }
  click() { this.clicks++; }
  async emit(name, event) { for (const fn of this.handlers[name] || []) await fn(event); }
}

async function setup(profile = "interactive-local", fail = false) {
  const nodes = new Map(), requests = [];
  const document = {
    getElementById(id) {
      if (!nodes.has(id)) nodes.set(id, new Element(id === "files" ? "input" : id === "profile" ? "select" : "div"));
      return nodes.get(id);
    },
    createElement(tag) { return new Element(tag); }
  };
  document.getElementById("profile").value = profile;
  class XHR {
    constructor() { this.headers = {}; this.upload = {}; }
    open(method, path) { this.method = method; this.path = path; }
    setRequestHeader(name, value) { this.headers[name] = value; }
    send(file) { this.file = file; requests.push(this); }
    finish() { this.status = fail ? 500 : 201; this.responseText = JSON.stringify(fail ? {error: "Upload fehlgeschlagen"} : {}); this.onload(); }
  }
  const ctx = vm.createContext({document, XMLHttpRequest: XHR, fetch: async () => ({ok: true, status: 200, json: async () => []}), console});
  vm.runInContext(fs.readFileSync("cmd/server/web/app.js", "utf8"), ctx);
  await new Promise(resolve => setImmediate(resolve));
  return {nodes, requests};
}

const tick = () => new Promise(resolve => setImmediate(resolve));

test("profile selection sits outside the clickable upload area", () => {
  const html = fs.readFileSync("cmd/server/web/index.html", "utf8");
  assert.match(html, /<select id="profile">/);
  assert.ok(html.indexOf('</select></div>') < html.indexOf('<div id="drop"'));
});

test("upload clicks ignore interactive controls and still open the file picker", async () => {
  const {nodes} = await setup(), drop = nodes.get("drop"), files = nodes.get("files");
  for (const tag of ["input", "select", "option", "label", "button", "a", "textarea"]) {
    await drop.emit("click", {target: new Element(tag)});
  }
  assert.equal(files.clicks, 0);
  await drop.emit("click", {target: drop});
  assert.equal(files.clicks, 1);
  let prevented = false;
  await drop.emit("keydown", {target: nodes.get("profile"), key: " ", preventDefault() { prevented = true; }});
  assert.equal(files.clicks, 1); assert.equal(prevented, false);
  await drop.emit("keydown", {target: drop, key: "Enter", preventDefault() { prevented = true; }});
  assert.equal(files.clicks, 2); assert.equal(prevented, true);
});

for (const profile of ["static", "interactive-local", "interactive-api"]) {
  test(`upload sends selected profile ${profile}`, async () => {
    const {nodes, requests} = await setup(profile);
    const done = nodes.get("files").onchange({target: {files: [{name: "demo.html"}]}});
    assert.equal(requests.length, 1);
    assert.equal(requests[0].headers["X-Security-Profile"], profile);
    assert.equal(requests[0].method, "POST"); assert.equal(requests[0].path, "/api/upload");
    assert.equal(nodes.get("profile").disabled, true);
    requests[0].finish(); await done;
    assert.equal(nodes.get("profile").disabled, false);
  });
}

test("a batch snapshots its profile and blocks file-picker clicks while uploading", async () => {
  const {nodes, requests} = await setup("static");
  const done = nodes.get("files").onchange({target: {files: [{name: "a.html"}, {name: "b.html"}]}});
  nodes.get("profile").value = "interactive-api";
  await nodes.get("drop").emit("click", {target: nodes.get("drop")});
  assert.equal(nodes.get("files").clicks, 0);
  requests[0].finish(); await tick();
  assert.equal(requests.length, 2); assert.equal(requests[1].headers["X-Security-Profile"], "static");
  requests[1].finish(); await done;
  assert.equal(nodes.get("profile").disabled, false);
});

test("drag-and-drop uses the selected profile and restores controls after failure", async () => {
  const {nodes, requests} = await setup("interactive-api", true);
  const done = nodes.get("drop").emit("drop", {dataTransfer: {files: [{name: "demo.htm"}]}, preventDefault() {}});
  await tick(); assert.equal(requests[0].headers["X-Security-Profile"], "interactive-api");
  requests[0].finish(); await done;
  assert.equal(nodes.get("profile").disabled, false); assert.equal(nodes.get("files").disabled, false);
  assert.match(nodes.get("status").textContent, /Upload fehlgeschlagen/);
});

test("invalid profiles do not start an upload", async () => {
  const {nodes, requests} = await setup("unknown");
  await nodes.get("files").onchange({target: {files: [{name: "demo.html"}]}});
  assert.equal(requests.length, 0); assert.equal(nodes.get("profile").disabled, false);
});

test("public list links have muted light and dark palettes", () => {
  const source = fs.readFileSync("cmd/server/main.go", "utf8");
  assert.match(source, /td:first-child a\{color:#475569/);
  assert.match(source, /td:first-child a\{color:#b8c3d4/);
  assert.match(source, /a:focus-visible/);
});
