package main

import "testing"

func TestPublicProfileBadgeMatchesAdmin(t *testing.T) {
	cases := []struct {
		profile, class, label string
	}{
		{"static", "badge-static", "Statisch"},
		{"interactive-local", "badge-interactive", "Interaktiv lokal"},
		{"interactive-api", "badge-api", "Interaktiv mit API"},
		{"", "badge-static", "Nicht hinterlegt (Altbestand)"},
		{"unknown", "badge-static", "Unbekannt (unknown)"},
		{"<img src=x onerror=alert(1)>", "badge-static", "Unbekannt (<img src=x onerror=alert(1)>)"},
	}
	for _, tc := range cases {
		t.Run(tc.profile, func(t *testing.T) {
			class, label := publicProfileBadge(SecurityProfile(tc.profile))
			if class != tc.class || label != tc.label {
				t.Fatalf("got (%q, %q), want (%q, %q)", class, label, tc.class, tc.label)
			}
		})
	}
}
