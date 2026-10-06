package main

import (
    "crypto/sha256"
    "fmt"
    "os"
    "path/filepath"
    "testing"
)
func TestBranding(t *testing.T) {
    a := testApp(t)
    for path, expected := range map[string]string{
        "/logo-light.png":"d98ccc9779f2c9bf0f8155bec14784952c9954484ae353824bcb345a41eb3fa3",
        "/logo-dark.png":"d529c003aff01ff22a4fe27f363b57c9770eb2abd7e89b5276fecc8e9feacd09",
    } {
        if w := request(a,"GET",path,"","","",false); w.Code != 401 { t.Fatalf("unauthenticated %s: %d",path,w.Code) }
        w := request(a,"GET",path,"","","",true)
        if w.Code != 200 || w.Header().Get("Content-Type") != "image/png" { t.Fatalf("%s: %d",path,w.Code) }
        if got := fmt.Sprintf("%x",sha256.Sum256(w.Body.Bytes())); got != expected { t.Fatalf("%s: checksum %s",path,got) }
        if head := request(a,"HEAD",path,"","","",true); head.Code != 200 || head.Body.Len() != 0 { t.Fatalf("HEAD %s",path) }
    }
}
func TestInitStorage(t *testing.T) {
    dir := t.TempDir()
    for i:=0; i<2; i++ { if err:=initStorage(dir); err!=nil { t.Fatal(err) } }
    for _,name:=range []string{"public","meta"} {
        info,err:=os.Stat(filepath.Join(dir,name))
        if err!=nil || !info.IsDir() { t.Fatalf("missing directory %s: %v",name,err) }
    }
}
