package main

import (
    "encoding/json"
    "mime"
    "os"
    "path/filepath"
    "strings"
    "testing"
)

func setLink(t *testing.T, a *app, id, slug string, code int) page {
    t.Helper()
    data, _ := json.Marshal(map[string]string{"slug":slug})
    w := request(a, "PUT", "/api/pages/"+id+"/link", "", string(data), a.origin, true)
    if w.Code != code { t.Fatalf("link: %d %s", w.Code, w.Body.String()) }
    var p page
    if code == 200 { if err := json.Unmarshal(w.Body.Bytes(), &p); err != nil { t.Fatal(err) } }
    return p
}
func TestCustomLinkLifecycle(t *testing.T) {
    a := testApp(t); p := publish(t, a)
    q := setLink(t, a, p.ID, "argocd-builder", 200)
    if q.URL != a.publicURL+"/pages/argocd-builder.html" || q.CanonicalURL != p.URL { t.Fatal(q) }
    first, err := os.Stat(filepath.Join(a.dir, "public", p.ID+".html")); if err != nil { t.Fatal(err) }
    alias, err := os.Stat(filepath.Join(a.dir, "public", "argocd-builder.html")); if err != nil || !os.SameFile(first, alias) { t.Fatal(err) }
    setLink(t, a, p.ID, "argocd-builder", 200)
    items, err := a.list(); if err != nil || len(items) != 1 || items[0].Slug != "argocd-builder" { t.Fatal(err, items) }
    setLink(t, a, p.ID, "builder-v2", 200)
    if _, err := os.Stat(filepath.Join(a.dir, "public", "argocd-builder.html")); !os.IsNotExist(err) { t.Fatal(err) }
    setLink(t, a, p.ID, "", 200)
    if _, err := os.Stat(filepath.Join(a.dir, "public", "builder-v2.html")); !os.IsNotExist(err) { t.Fatal(err) }
    setLink(t, a, p.ID, "delete-me", 200)
    if w := request(a, "DELETE", "/api/pages/"+p.ID, "", "", a.origin, true); w.Code != 204 { t.Fatal(w.Code) }
    if _, err := os.Stat(filepath.Join(a.dir, "public", "delete-me.html")); !os.IsNotExist(err) { t.Fatal(err) }
}
func TestCustomLinkConflictAndValidation(t *testing.T) {
    a := testApp(t); p, q := publish(t,a), publish(t,a)
    setLink(t,a,p.ID,"shared",200); setLink(t,a,q.ID,"shared",409)
    for _, slug := range []string{"../x", "a/b", "Upper", "a.html", "-a", "a-", strings.Repeat("x",65), p.ID} { setLink(t,a,p.ID,slug,400) }
    for _, body := range []string{`{}`, `{"slug":null}`, `{"slug":"valid","extra":1}`, `{"slug":"valid"} {}`} {
        if w := request(a,"PUT","/api/pages/"+p.ID+"/link","",body,a.origin,true); w.Code != 400 { t.Fatal(body,w.Code) }
    }
    if w := request(a,"PUT","/api/pages/"+p.ID+"/link","",strings.Repeat("x",1025),a.origin,true); w.Code != 413 { t.Fatal(w.Code) }
    for _, auth := range []bool{false,true} {
        w := request(a,"PUT","/api/pages/"+p.ID+"/link","",`{"slug":"evil"}`,"http://evil.test",auth)
        expected:=401; if auth { expected=403 }; if w.Code!=expected { t.Fatal(w.Code) }
    }
}
func TestDownload(t *testing.T) {
    a:=testApp(t); p:=publish(t,a)
    for _, method:=range []string{"GET","HEAD"} {
        w:=request(a,method,"/api/pages/"+p.ID+"/download","","","",true)
        disposition,params,err:=mime.ParseMediaType(w.Header().Get("Content-Disposition"))
        if w.Code!=200 || err!=nil || disposition!="attachment" || params["filename"]!=p.Name || w.Header().Get("Content-Type")!="application/octet-stream" { t.Fatal(w.Code,err,w.Header()) }
        if method=="GET" { b,err:=os.ReadFile(filepath.Join(a.dir,"public",p.ID+".html")); if err!=nil || string(b)!=w.Body.String() { t.Fatal(err) } } else if w.Body.Len()!=0 { t.Fatal("HEAD body") }
    }
    if w:=request(a,"GET","/api/pages/"+p.ID+"/download","","","",false); w.Code!=401 { t.Fatal(w.Code) }
    if w:=request(a,"GET","/api/pages/not-an-id/download","","","",true); w.Code!=400 { t.Fatal(w.Code) }
    if w:=request(a,"GET","/api/pages/"+strings.Repeat("f",32)+"/download","","","",true); w.Code!=404 { t.Fatal(w.Code) }
}
