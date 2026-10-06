const test = require("node:test");
const assert = require("node:assert/strict");
const fs = require("node:fs");
const vm = require("node:vm");
class Element {
  constructor() { this.children=[]; this.value=""; this.textContent=""; this.hidden=false; this.classList={add(){},remove(){}}; }
  append(x) { this.children.push(x); }
  replaceChildren(...x) { this.children=x; }
  addEventListener() {}
}
async function context(answer, conflict=false) {
  const nodes=new Map(), calls=[];
  let p={id:"a".repeat(32),name:"Demo ä.html",size:12,created:"2026-10-06",slug:"old-name",url:"https://pages.example.test/pages/old-name.html"};
  const document={getElementById(id){if(!nodes.has(id))nodes.set(id,new Element());return nodes.get(id);},createElement(){return new Element();}};
  const ctx=vm.createContext({document,navigator:{clipboard:{writeText:async()=>{}}},prompt:()=>answer,confirm:()=>false,console,fetch:async(path,options={})=>{
    calls.push({path,options});
    if(options.method==="PUT") {
      if(conflict)return {ok:false,status:409,json:async()=>({error:"URL-Name bereits belegt"})};
      p={...p,slug:JSON.parse(options.body).slug};p.url=`https://pages.example.test/pages/${p.slug||p.id}.html`;
      return {ok:true,status:200,json:async()=>p};
    }
    return {ok:true,status:200,json:async()=>[p]};
  }});
  vm.runInContext(fs.readFileSync("cmd/server/web/app.js","utf8"),ctx);
  await new Promise(resolve=>setImmediate(resolve));calls.length=0;
  const action=name=>nodes.get("pages").children[0].children[4].children.find(x=>x.textContent===name);
  return {ctx,nodes,calls,action};
}
test("download uses authenticated API and original filename",async()=>{
  const {action}=await context(null),link=action("Herunterladen");
  assert.equal(link.href,`/api/pages/${"a".repeat(32)}/download`);assert.equal(link.download,"Demo ä.html");
});
test("link edit sends PUT JSON and refreshes the list",async()=>{
  const {action,calls,nodes}=await context("argocd-builder");await action("Link ändern").onclick();
  assert.equal(calls[0].options.method,"PUT");assert.equal(calls[0].options.credentials,"same-origin");
  assert.equal(calls[0].options.body,'{"slug":"argocd-builder"}');assert.equal(calls[1].path,"/api/pages");
  assert.match(nodes.get("status").textContent,/argocd-builder\.html/);
});
test("empty input resets the alias",async()=>{
  const {action,calls}=await context("");await action("Link ändern").onclick();assert.equal(calls[0].options.body,'{"slug":""}');
});
test("cancelled edit sends no request",async()=>{
  const {action,calls}=await context(null);await action("Link ändern").onclick();assert.equal(calls.length,0);
});
test("invalid aliases are rejected before a request",async()=>{
  for(const value of ["../x","Upper","a.html","-a","a-","x".repeat(65),"f".repeat(32)]) {
    const {action,calls,nodes}=await context(value);await action("Link ändern").onclick();assert.equal(calls.length,0);assert.match(nodes.get("status").textContent,/Ungültiger/);
  }
});
test("conflict is shown and button is re-enabled",async()=>{
  const {action,nodes}=await context("shared",true),button=action("Link ändern");await button.onclick();
  assert.equal(button.disabled,false);assert.match(nodes.get("status").textContent,/bereits belegt/);
});
test("cancelled deletion and opener isolation are retained",async()=>{
  const {action,calls}=await context(null);await action("Löschen").onclick();assert.equal(calls.length,0);
  assert.equal(action("Öffnen").rel,"noopener noreferrer");assert.equal(action("Öffnen").target,"_blank");
});
