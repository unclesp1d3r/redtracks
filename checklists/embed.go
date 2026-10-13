// Package checklists embeds the shipped sample checklist definitions so the
// app can install them on first launch without a network fetch.
package checklists

import "embed"

// FS holds the shipped checklist JSON files and their readme.
//
//go:embed *.json _readme.md
var FS embed.FS
