# Sicherheitsrichtlinie

Bitte keine Passwörter, hochgeladenen vertraulichen Dokumente oder ausnutzbaren Details in öffentlichen Issues veröffentlichen. Sicherheitsberichte an den privat definierten Maintainer-Kontakt des eigenen Repositories senden; diesen vor einer Veröffentlichung hinterlegen.

Die Anwendung ist für vertrauenswürdige Administratoren und öffentlich lesbare HTML-Dateien konzipiert. Randomisierte IDs sind keine Zugriffsberechtigung. Alle publizierten Seiten sind öffentlich. Es gibt keine Benutzerverwaltung, kein OIDC, keine automatische Inhaltsbereinigung und keinen Virenscanner. Basic Authentication nur lokal oder über HTTPS verwenden.

Bei einer Schwachstelle: Veröffentlichung stoppen, Zugangsdaten wechseln, betroffene Images aktualisieren, erneut scannen, testen und ausrollen. Keine pauschalen CVE-Ausnahmen verwenden; Ausnahmen mit Begründung und Ablaufdatum dokumentieren.

Der vorgeschlagene CVE-Gate scannt die endgültigen Runtime-Images. Er ist weder ein Penetrationstest noch ein Nachweis vollständiger CVE-Freiheit. Build-Toolchain und GitHub Actions zusätzlich absichern; Action-Refs vor produktivem Einsatz auf geprüfte Commit-SHAs pinnen.
