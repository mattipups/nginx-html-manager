const test = require("node:test");
const assert = require("node:assert/strict");
const fs = require("node:fs");
const vm = require("node:vm");
class Element {
  constructor() { this.children = []; this.value = ""; this.textContent = ""; this.hidden = false; this.listeners = {}; this.classList = {add(){},remove(){}}; }
  append(x) { this.children.push(x); }
  replaceChildren(...x) { this.children = x; }
  addEventListener(event, fn) { this.listeners[event] = fn; }
  click() { if (this.onclick) this.onclick({target:this}); }
}
async function context() {
  const nodes = new Map();
  let networkUploads = 0;
  const document = {getElementById(id) { if (!nodes.has(id)) nodes.set(id,new Element()); return nodes.get(id); }, createElement() { return new Element(); }};
  document.getElementById("profile").value = "interactive-local";
  const ctx = vm.createContext({XMLHttpRequest: class { constructor() { networkUploads++; throw new Error("Unexpected network upload"); } }, document, navigator:{clipboard:{writeText:async()=>{}}}, fetch:async()=>({ok:true,status:200,json:async()=>[]}), confirm:()=>false, console});
  vm.runInContext(fs.readFileSync("cmd/server/web/app.js","utf8"),ctx);
  await new Promise(resolve=>setImmediate(resolve));
  return {ctx,nodes,networkUploads:()=>networkUploads};
}
test("empty list is shown",async()=>{const {nodes}=await context();assert.equal(nodes.get("empty").hidden,false);});
test("filenames are text, never innerHTML",async()=>{
  const {ctx,nodes}=await context();
  vm.runInContext(`pages=[{id:"a",name:"<img src=x onerror=alert(1)>.html",size:123,created:"2026-01-01",url:"http://localhost:8081/pages/a.html"}];render();`,ctx);
  const cell=nodes.get("pages").children[0].children[0];
  assert.equal(cell.textContent,"<img src=x onerror=alert(1)>.html");assert.equal(cell.innerHTML,undefined);
});
test("external tab cannot access the opener",async()=>{
  const {ctx,nodes}=await context();vm.runInContext(`pages=[{id:"a",name:"a.html",size:1,created:"2026-01-01",url:"http://localhost:8081/pages/a.html"}];render();`,ctx);
  const link=nodes.get("pages").children[0].children[4].children[0];assert.equal(link.rel,"noopener noreferrer");assert.equal(link.target,"_blank");
});
test("name filtering is case insensitive",async()=>{
  const {ctx,nodes}=await context();nodes.get("search").value="DEMO";
  vm.runInContext(`pages=[{name:"demo.html",size:1,created:"2026-01-01",url:"http://localhost:8081/x"},{name:"other.html",size:1,created:"2026-01-01",url:"http://localhost:8081/y"}];render();`,ctx);
  assert.equal(nodes.get("pages").children.length,1);
});
test("unsupported extension fails without a network upload",async()=>{
  const {ctx,nodes,networkUploads}=await context();await vm.runInContext(`upload([{name:"shell.php"}])`,ctx);
  assert.match(nodes.get("status").textContent,/Nur .html\/.htm erlaubt/);assert.equal(nodes.get("files").disabled,false);
  assert.equal(networkUploads(),0);assert.equal(nodes.get("profile").disabled,false);
});
test("cancelled delete does not request deletion",async()=>{
  const {ctx,nodes}=await context();vm.runInContext(`pages=[{id:"a",name:"a.html",size:1,created:"2026-01-01",url:"http://localhost:8081/x"}];render();`,ctx);
  const remove=nodes.get("pages").children[0].children[4].children[2];await remove.onclick();assert.notEqual(remove.disabled,true);
});

test("stored security profiles are shown independently of the upload selection",async()=>{
  const {ctx,nodes}=await context();nodes.get("profile").value="static";
  vm.runInContext(`pages=["static","interactive-local","interactive-api"].map((profile,i)=>({id:String(i),name:"demo.html",size:1,created:"2026-01-01",url:"http://localhost:8081/x",profile}));render();`,ctx);
  assert.deepEqual(nodes.get("pages").children.map(row=>row.children[3].children[0].textContent),["Statisch","Interaktiv lokal","Interaktiv mit API"]);
  for(const row of nodes.get("pages").children) assert.equal(row.children.length,5);
});
test("legacy pages without a stored profile are explicitly marked",async()=>{
  const {ctx,nodes}=await context();
  vm.runInContext(`pages=[undefined,null,""].map(profile=>({name:"legacy.html",size:1,created:"2026-01-01",profile}));render();`,ctx);
  for(const row of nodes.get("pages").children) assert.equal(row.children[3].children[0].textContent,"Nicht hinterlegt (Altbestand)");
});
test("unknown security profile values are rendered as text",async()=>{
  const {ctx,nodes}=await context();
  vm.runInContext(`pages=[{name:"demo.html",size:1,created:"2026-01-01",profile:"<img src=x onerror=alert(1)>"}];render();`,ctx);
  const cell=nodes.get("pages").children[0].children[3].children[0];
  assert.equal(cell.textContent,"Unbekannt (<img src=x onerror=alert(1)>)");assert.equal(cell.innerHTML,undefined);
});
test("admin table header includes the security profile column",()=>{
  assert.match(fs.readFileSync("cmd/server/web/index.html","utf8"),/<th>Veröffentlicht<\/th><th>Sicherheitsprofil<\/th><th>Aktionen<\/th>/);
});

test("profile badges use matching classes without trusting metadata as CSS",async()=>{
  const {ctx,nodes}=await context();
  vm.runInContext(`pages=["static","interactive-local","interactive-api",undefined,"unexpected-class"].map(profile=>({name:"demo.html",size:1,created:"2026-01-01",profile}));render();`,ctx);
  assert.deepEqual(nodes.get("pages").children.map(row=>row.children[3].children[0].className),["badge badge-static","badge badge-interactive","badge badge-api","badge badge-static","badge badge-static"]);
  for(const row of nodes.get("pages").children) assert.equal(row.children[3].children.length,1);
});
test("profile badge dimensions and light palette match the public overview",()=>{
  const css=fs.readFileSync("cmd/server/web/style.css","utf8");
  for(const rule of [".badge{display:inline-block;padding:2px 8px;border-radius:4px;font-size:12px;font-weight:600}",".badge-static{background:#e2e8f0;color:#334155}",".badge-interactive{background:#dbeafe;color:#1e40af}",".badge-api{background:#fef3c7;color:#92400e}"]) assert.ok(css.includes(rule),rule);
});
test("profile badges include the public overview dark palette",()=>{
  const css=fs.readFileSync("cmd/server/web/style.css","utf8");
  assert.ok(css.includes("@media(prefers-color-scheme:dark){\n.badge-static"));
  for(const rule of [".badge-static{background:#374151;color:#e5e7eb}",".badge-interactive{background:#1e3a5f;color:#bfdbfe}",".badge-api{background:#451a03;color:#fde68a}"]) assert.ok(css.includes(rule),rule);
});
