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
  const document = {getElementById(id) { if (!nodes.has(id)) nodes.set(id,new Element()); return nodes.get(id); }, createElement() { return new Element(); }};
  const ctx = vm.createContext({document, navigator:{clipboard:{writeText:async()=>{}}}, fetch:async()=>({ok:true,status:200,json:async()=>[]}), confirm:()=>false, console});
  vm.runInContext(fs.readFileSync("cmd/server/web/app.js","utf8"),ctx);
  await new Promise(resolve=>setImmediate(resolve));
  return {ctx,nodes};
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
  const link=nodes.get("pages").children[0].children[3].children[0];assert.equal(link.rel,"noopener noreferrer");assert.equal(link.target,"_blank");
});
test("name filtering is case insensitive",async()=>{
  const {ctx,nodes}=await context();nodes.get("search").value="DEMO";
  vm.runInContext(`pages=[{name:"demo.html",size:1,created:"2026-01-01",url:"http://localhost:8081/x"},{name:"other.html",size:1,created:"2026-01-01",url:"http://localhost:8081/y"}];render();`,ctx);
  assert.equal(nodes.get("pages").children.length,1);
});
test("unsupported extension fails without a network upload",async()=>{
  const {ctx,nodes}=await context();await vm.runInContext(`upload([{name:"shell.php"}])`,ctx);
  assert.match(nodes.get("status").textContent,/Nur .html\/.htm erlaubt/);assert.equal(nodes.get("files").disabled,false);
});
test("cancelled delete does not request deletion",async()=>{
  const {ctx,nodes}=await context();vm.runInContext(`pages=[{id:"a",name:"a.html",size:1,created:"2026-01-01",url:"http://localhost:8081/x"}];render();`,ctx);
  const remove=nodes.get("pages").children[0].children[3].children[2];await remove.onclick();assert.notEqual(remove.disabled,true);
});
