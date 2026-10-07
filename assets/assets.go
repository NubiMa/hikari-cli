// Package assets embeds static data files (personas, themes, ascii art)
// into the binary so that Hikari works correctly when installed via
// `go install` or distributed as a prebuilt binary — with no dependency
// on the source tree being present at runtime.
package assets

import "embed"

// Personas contains the bundled built-in persona YAML files.
//
//go:embed personas/*.yaml
var Personas embed.FS

// Themes contains the bundled built-in theme TOML files.
//
//go:embed themes/*.toml
var Themes embed.FS

// ASCII contains the bundled ASCII art banner text files.
//
//go:embed ascii/*.txt
var ASCII embed.FS
