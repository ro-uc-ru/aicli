// Package templates embeds the files of the generated FastAPI project
// skeleton. Files with the .tmpl suffix are rendered with the project name;
// all others are copied verbatim.
package templates

import "embed"

// Skeleton holds the base project skeleton rooted at the "skeleton" directory.
// It is read-only and includes dotfiles (.env.example, .gitignore, .gitkeep).
//
//go:embed all:skeleton
var Skeleton embed.FS

// LLMProvider holds the templates of the `aicli add llm-provider` feature
// (spec 005), rooted at the "llmprovider" directory.
//
//go:embed all:llmprovider
var LLMProvider embed.FS
