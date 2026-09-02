package handler

import (
	"html/template"
	"net/http"

	"github.com/jonradoff/flipbook/internal/config"
	"github.com/jonradoff/flipbook/internal/server"
	"github.com/jonradoff/flipbook/internal/storage"
)

var handler http.Handler

func init() {
	cfg := config.Load()
	store := storage.New("/tmp/flipbooks")
	var tmpl *template.Template
	tmpl = server.ParseTemplates()
	handler = server.BuildHandler(cfg, nil, store, nil, tmpl)
}

// Handler is the Vercel serverless entry point.
func Handler(w http.ResponseWriter, r *http.Request) {
	handler.ServeHTTP(w, r)
}
