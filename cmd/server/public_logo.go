package main

import (
    "encoding/base64"
    "fmt"

    branding "example.com/nginx-html-manager/logo/logos"
)

func publicLogoPicture() (string, error) {
    light, err := branding.Assets.ReadFile("builder-light-1400x700.png")
    if err != nil { return "", err }
    dark, err := branding.Assets.ReadFile("builder-dark-1400x700.png")
    if err != nil { return "", err }
    return fmt.Sprintf(`<picture class="brand"><source srcset="data:image/png;base64,%s" media="(prefers-color-scheme: dark)"><img src="data:image/png;base64,%s" alt="Builder" width="1400" height="700"></picture>`+"\n", base64.StdEncoding.EncodeToString(dark), base64.StdEncoding.EncodeToString(light)), nil
}
