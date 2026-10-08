package main

import "fmt"

// Match the stored-profile labels and badge classes used by the admin UI.
func publicProfileBadge(profile SecurityProfile) (string, string) {
	switch string(profile) {
	case "static":
		return "badge-static", "Statisch"
	case "interactive-local":
		return "badge-interactive", "Interaktiv lokal"
	case "interactive-api":
		return "badge-api", "Interaktiv mit API"
	case "":
		return "badge-static", "Nicht hinterlegt (Altbestand)"
	default:
		return "badge-static", fmt.Sprintf("Unbekannt (%s)", profile)
	}
}
