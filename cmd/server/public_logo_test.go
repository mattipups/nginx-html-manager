package main

import (
    "bytes"
    "encoding/base64"
    "os"
    "path/filepath"
    "regexp"
    "strings"
    "testing"

    branding "example.com/nginx-html-manager/logo/logos"
)

func TestPublicLogoUsesSameLightAndDarkAssets(t *testing.T) {
    picture, err := publicLogoPicture()
    if err != nil { t.Fatal(err) }
    matches := regexp.MustCompile(`data:image/png;base64,([A-Za-z0-9+/=]+)`).FindAllStringSubmatch(picture,-1)
    if len(matches) != 2 { t.Fatal("both logo variants required") }
    for i, name := range []string{"builder-dark-1400x700.png","builder-light-1400x700.png"} {
        data, err := base64.StdEncoding.DecodeString(matches[i][1])
        if err != nil { t.Fatal(err) }
        expected, err := branding.Assets.ReadFile(name)
        if err != nil { t.Fatal(err) }
        if !bytes.Equal(data,expected) { t.Fatal("public and admin logos differ") }
    }
    if !strings.Contains(picture,`media="(prefers-color-scheme: dark)"`) { t.Fatal("dark theme missing") }
}

func TestPublicIndexIncludesLogoBeforeHeading(t *testing.T) {
    a := testApp(t)
    if err := a.renderPublicIndex(); err != nil { t.Fatal(err) }
    data, err := os.ReadFile(filepath.Join(a.dir,"public","index.html"))
    if err != nil { t.Fatal(err) }
    document := string(data)
    picture, err := publicLogoPicture()
    if err != nil { t.Fatal(err) }
    start := strings.Index(document,picture)
    heading := strings.Index(document,"<h1>")
    if start < 0 || heading < 0 || start > heading { t.Fatal("public logo missing before heading") }
    if !strings.Contains(document,"HTML-PUBLISHER · 1.5.0") { t.Fatal("public version missing") }
}
