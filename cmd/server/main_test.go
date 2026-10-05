package main

import (
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "net/url"
    "os"
    "path/filepath"
    "strings"
    "testing"
)
func testApp(t *testing.T) *app {
    t.Helper()
    a := &app{dir:t.TempDir(), user:"admin", password:"test-password-long-enough", origin:"http://localhost:8080", publicURL:"http://localhost:8081", maxBytes:1024, maxPages:5, uploadSlots:make(chan struct{},1)}
    for _, name := range []string{"meta","public"} {
        if err := os.Mkdir(filepath.Join(a.dir,name),0755); err != nil { t.Fatal(err) }
    }
    return a
}
func request(a *app, method, path, filename, body, origin string, auth bool) *httptest.ResponseRecorder {
    r := httptest.NewRequest(method,path,strings.NewReader(body))
    if filename != "" { r.Header.Set("X-File-Name",url.PathEscape(filename)) }
    if origin != "" { r.Header.Set("Origin",origin) }
    if auth { r.SetBasicAuth(a.user,a.password) }
    w := httptest.NewRecorder()
    a.handler().ServeHTTP(w,r)
    return w
}
func publish(t *testing.T, a *app) page {
    t.Helper()
    w := request(a,"POST","/api/upload","demo.html","<!doctype html><html><body>Hello</body></html>",a.origin,true)
    if w.Code != 201 { t.Fatalf("upload: %d %s",w.Code,w.Body.String()) }
    var p page
    if err := json.Unmarshal(w.Body.Bytes(),&p); err != nil { t.Fatal(err) }
    return p
}
func TestAuthentication(t *testing.T) {
    a := testApp(t)
    for _, path := range []string{"/","/app.js","/api/pages"} {
        if w := request(a,"GET",path,"","","",false); w.Code != 401 { t.Fatalf("%s: %d",path,w.Code) }
    }
    r := httptest.NewRequest("GET","/",nil)
    r.SetBasicAuth("admin","incorrect")
    w := httptest.NewRecorder()
    a.handler().ServeHTTP(w,r)
    if w.Code != 401 { t.Fatal(w.Code) }
}
func TestOriginProtection(t *testing.T) {
    a := testApp(t)
    for _, origin := range []string{"","http://evil.test","http://localhost:8081","null"} {
        w := request(a,"POST","/api/upload","x.html","<!doctype html>",origin,true)
        if w.Code != 403 { t.Fatalf("origin %q: %d",origin,w.Code) }
    }
}
func TestUploadAndList(t *testing.T) {
    a := testApp(t)
    p := publish(t,a)
    if !idRE.MatchString(p.ID) || len(p.SHA256) != 64 || p.URL != a.publicURL+"/pages/"+p.ID+".html" { t.Fatalf("bad metadata: %+v",p) }
    b, err := os.ReadFile(filepath.Join(a.dir,"public",p.ID+".html"))
    if err != nil || !strings.Contains(string(b),"Hello") { t.Fatal(err,string(b)) }
    w := request(a,"GET","/api/pages","","","",true)
    var pages []page
    if err := json.Unmarshal(w.Body.Bytes(),&pages); err != nil { t.Fatal(err) }
    if w.Code != 200 || len(pages) != 1 || pages[0].Name != "demo.html" { t.Fatal(w.Code,pages) }
}
func TestFileNameValidation(t *testing.T) {
    a := testApp(t)
    for _, name := range []string{"../x.html",`a\b.html`,"x.php","x.html.exe","x\n.html",strings.Repeat("a",181)+".html"} {
        w := request(a,"POST","/api/upload",name,"<!doctype html>",a.origin,true)
        if w.Code != 400 { t.Fatalf("name %q: %d",name,w.Code) }
    }
    for _, name := range []string{"a.htm","A.HTML","Hallo Welt.html","ä.html"} {
        if !validName(name) { t.Fatal(name) }
    }
}
func TestContentValidation(t *testing.T) {
    a := testApp(t)
    for _, body := range []string{"","hello","\x00<html>","<html>\xff"} {
        w := request(a,"POST","/api/upload","x.html",body,a.origin,true)
        if w.Code != 400 { t.Fatalf("content %q: %d",body,w.Code) }
    }
}
func TestSizeLimit(t *testing.T) {
    a := testApp(t)
    w := request(a,"POST","/api/upload","x.html","<html>"+strings.Repeat("a",1024),a.origin,true)
    if w.Code != 413 { t.Fatal(w.Code) }
}
func TestDuplicateNamesUseDistinctIDs(t *testing.T) {
    a := testApp(t)
    p, q := publish(t,a),publish(t,a)
    if p.ID == q.ID { t.Fatal("IDs must differ") }
    items,err := a.list()
    if err != nil || len(items) != 2 { t.Fatal(err,items) }
}
func TestPageLimit(t *testing.T) {
    a := testApp(t)
    a.maxPages=1
    publish(t,a)
    w := request(a,"POST","/api/upload","x.html","<html>",a.origin,true)
    if w.Code != 409 { t.Fatal(w.Code) }
}
func TestDelete(t *testing.T) {
    a := testApp(t)
    p := publish(t,a)
    w := request(a,"DELETE","/api/pages/"+p.ID,"","",a.origin,true)
    if w.Code != 204 { t.Fatal(w.Code,w.Body.String()) }
    for _, path := range []string{filepath.Join(a.dir,"public",p.ID+".html"),filepath.Join(a.dir,"meta",p.ID+".json")} {
        if _,err := os.Stat(path); !os.IsNotExist(err) { t.Fatal(path,err) }
    }
    if w := request(a,"DELETE","/api/pages/"+p.ID,"","",a.origin,true); w.Code != 404 { t.Fatal(w.Code) }
}
func TestDeleteRejectsTraversal(t *testing.T) {
    a := testApp(t)
    if w := request(a,"DELETE","/api/pages/../../x","","",a.origin,true); w.Code != 400 { t.Fatal(w.Code) }
}
func TestStaticRoutesAndHealth(t *testing.T) {
    a := testApp(t)
    for _, path := range []string{"/","/app.js","/style.css"} {
        w := request(a,"GET",path,"","","",true)
        if w.Code != 200 || w.Header().Get("Content-Security-Policy") == "" { t.Fatal(path,w.Code) }
    }
    if w := request(a,"GET","/healthz","","","",false); w.Code != 200 { t.Fatal(w.Code) }
    if w := request(a,"GET","/web/index.html","","","",true); w.Code != 404 { t.Fatal(w.Code) }
}
func TestAtomicWrite(t *testing.T) {
    dir:=t.TempDir()
    if err:=atomicWrite(dir,"page.html",[]byte("content")); err != nil { t.Fatal(err) }
    entries,err:=os.ReadDir(dir)
    if err != nil || len(entries)!=1 || entries[0].Name()!="page.html" { t.Fatal(entries,err) }
}
func TestValidOrigin(t *testing.T) {
    for _, origin:=range []string{"https://admin.example.test","http://localhost:8080"} {
        if !validOrigin(origin) { t.Fatal(origin) }
    }
    for _, origin:=range []string{"","javascript:alert(1)","https://example.test/","https://u:p@example.test","https://example.test?q=x"} {
        if validOrigin(origin) { t.Fatal(origin) }
    }
}
func TestMethods(t *testing.T) {
    a:=testApp(t)
    if w:=request(a,http.MethodPut,"/api/upload","","",a.origin,true); w.Code!=405 { t.Fatal(w.Code) }
}
