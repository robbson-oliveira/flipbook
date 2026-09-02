// Package handler is the Vercel serverless function entry point.
// All traffic is routed through the same chi router as the standalone server
// via a lazy-initialized singleton (sync.Once) so the MongoDB connection
// and parsed templates are reused across warm invocations.
package handler

import (
	"context"
	"log"
	"net/http"
	"path/filepath"
	"sync"

	"github.com/jonradoff/flipbook/internal/config"
	"github.com/jonradoff/flipbook/internal/database"
	"github.com/jonradoff/flipbook/internal/server"
	"github.com/jonradoff/flipbook/internal/storage"
)

var (
	once       sync.Once
	appHandler http.Handler
)

// Handler is the Vercel Go serverless function entry point.
func Handler(w http.ResponseWriter, r *http.Request) {
	once.Do(func() {
		appHandler = buildVercelHandler()
	})
	appHandler.ServeHTTP(w, r)
}

func buildVercelHandler() http.Handler {
	cfg := config.Load()

	db, err := database.Open(context.Background(), cfg.MongoURI, cfg.MongoDB)
	if err != nil {
		log.Printf("[vercel] MongoDB connection failed: %v", err)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w,
				"Database unavailable. Set the FLIPBOOK_MONGO_URI environment variable in your Vercel project settings.",
				http.StatusServiceUnavailable,
			)
		})
	}

	// /tmp is the only writable directory on Vercel.
	// Page images are served via GridFS fallback when not on disk.
	store := storage.New(filepath.Join("/tmp", "flipbooks"))

	tmpl := server.ParseTemplates()

	// nil worker = read-only mode (no background conversion on Vercel)
	return server.BuildHandler(cfg, db, store, nil, tmpl)
}
