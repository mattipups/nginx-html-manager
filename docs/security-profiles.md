# Sicherheitsarchitektur und Profil-Isolation (v1.4.0)

## Problemstellung: Pfadbasierte URLs vs. Origin-Isolation

Im Web-Sicherheitsmodell ist der Browser-Speicher (`localStorage`, `sessionStorage`, `IndexedDB`, Cookies) an den **Origin** gebunden:
`Origin = <Schema>://<Host>:<Port>`

Werden mehrere Webanwendungen (z. B. Builder A und Builder B) über unterschiedliche Pfade desselben Hosts bereitgestellt (`pages.example.test/pages/app1.html` und `pages.example.test/pages/app2.html`), teilen sie denselben Browser-Ursprung. Eine kompromittierte oder fehlerhafte Seite kann den LocalStorage anderer Seiten auslesen oder überschreiben.

Gemäß den **OWASP File Upload Security Cheat Sheets** müssen von Benutzern hochgeladene Inhalte auf einem separaten, isolierten Host bzw. Subdomain ausgeliefert werden.

## Die drei Sicherheitsprofile

| Profil | Gedachte Verwendung | CSP Freigaben |
|---|---|---|
| **Statisch** (`static`) | Dokumentation, Reports, reine HTML/CSS-Seiten | `sandbox; default-src 'none'; style-src 'unsafe-inline'; img-src data:; base-uri 'none'; form-action 'none'; frame-ancestors 'none'` |
| **Interaktiv lokal** (`interactive-local`) | Generatoren, Konfiguratoren, Offline-Tools | `sandbox allow-scripts allow-same-origin allow-downloads allow-modals allow-popups allow-popups-to-escape-sandbox; default-src 'none'; script-src 'unsafe-inline'; style-src 'unsafe-inline'; img-src data: blob:; font-src data:; connect-src 'none'; base-uri 'none'; form-action 'none'` |
| **Interaktiv mit API** (`interactive-api`) | ArgoCD-Builder mit Git-Commit, externe Webhooks | `sandbox allow-scripts allow-same-origin allow-downloads allow-modals allow-popups allow-popups-to-escape-sandbox; default-src 'none'; script-src 'unsafe-inline'; style-src 'unsafe-inline'; img-src data: blob:; font-src data:; connect-src https:; base-uri 'none'; form-action 'none'` |

## Echte Origin-Isolation durch Subdomains

In v1.4.0 erhält jede interaktive Anwendung ihre eigene Subdomain:
- `http://<slug>.localhost:8081/` (lokale Umgebung)
- `https://<slug>.pages.example.test/` (Produktion / Kubernetes Ingress)

Dadurch besitzt jede Anwendung einen vollständig isolierten `localStorage`-Namespace. Presets, Cache und Anwendungsdaten von App A sind für App B technisch unerreichbar.
