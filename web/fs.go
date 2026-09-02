package web

import "embed"

// FS embeds all templates and static files into the binary.
// This ensures they are always available regardless of working directory,
// which is required for Vercel serverless deployments.
//
//go:embed templates static
var FS embed.FS
