// Package recipebank exposes repository documents bundled into the app.
package recipebank

import _ "embed"

// Changelog is CHANGELOG.md, bundled so "What's new" works offline and shows
// the notes for the exact version that is running.
//
//go:embed CHANGELOG.md
var Changelog string
