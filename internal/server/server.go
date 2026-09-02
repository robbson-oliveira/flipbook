// Package server provides the shared HTTP handler factory used by both
// the standalone server (main.go) and the Vercel serverless adapter (api/index.go).
package server

import (
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jonradoff/flipbook/internal/auth"
	"github.com/jonradoff/flipbook/internal/config"
	"github.com/jonradoff/flipbook/internal/database"
	"github.com/jonradoff/flipbook/internal/handlers"
	"github.com/jonradoff/flipbook/internal/storage"
	"github.com/jonradoff/flipbook/internal/worker"
	web "github.com/jonradoff/flipbook/web"
)

// ParseTemplates parses all HTML templates from the embedded web.FS.
func ParseTemplates() *template.Template {
	funcMap := template.FuncMap{
		"last": func(i, total int) bool { return i == total-1 },
		"add":  func(a, b int) int { return a + b },
	}
	tmpl, err := template.New("").Funcs(funcMap).ParseFS(web.FS,
		"templates/*.html",
		"templates/admin/*.html",
	)
	if err != nil {
		log.Fatalf("Failed to parse templates: %v", err)
	}
	return tmpl
}

// BuildHandler constructs the full HTTP handler (router + middleware).
// Pass w=nil to disable the background worker (e.g. on Vercel — read-only mode).
func BuildHandler(cfg *config.Config, db *database.DB, store *storage.Storage, w *worker.Worker, tmpl *template.Template) http.Handler {
	// Initialize auth
	a := auth.New(db, cfg.SessionSecret, tmpl)

	if !a.HasPassword() {
		log.Println("WARNING: No admin password set. Run './flipbook set-password' to secure the admin area.")
	}

	// Setup handlers
	adminH := handlers.NewAdminHandler(db, store, w, tmpl, cfg.BaseURL)
	viewerH := handlers.NewViewerHandler(db, store, tmpl, cfg.BaseURL)
	embedH := handlers.NewEmbedHandler(db, store, tmpl, cfg.BaseURL)
	apiH := handlers.NewAPIHandler(db, store, w, cfg.BaseURL)

	// Static files from embedded FS
	staticFS, _ := fs.Sub(web.FS, "static")

	// Setup router
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Compress(5))
	r.Use(SecurityHeaders)

	// Static files (served from embedded FS)
	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))

	// Serve flipbook page images — filesystem first, then GridFS fallback (for serverless).
	r.Get("/data/flipbooks/{id}/pages/{filename}", func(rw http.ResponseWriter, req *http.Request) {
		rw.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		id := filepath.Base(chi.URLParam(req, "id"))
		fname := filepath.Base(chi.URLParam(req, "filename"))
		diskPath := filepath.Join(cfg.DataDir, "flipbooks", id, "pages", fname)
		if _, err := os.Stat(diskPath); err == nil {
			http.ServeFile(rw, req, diskPath)
			return
		}
		rw.Header().Set("Content-Type", "image/png")
		if err := db.StreamPageImage(req.Context(), id, "pages/"+fname, rw); err != nil {
			http.NotFound(rw, req)
		}
	})
	r.Get("/data/flipbooks/{id}/thumbs/{filename}", func(rw http.ResponseWriter, req *http.Request) {
		rw.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		id := filepath.Base(chi.URLParam(req, "id"))
		fname := filepath.Base(chi.URLParam(req, "filename"))
		diskPath := filepath.Join(cfg.DataDir, "flipbooks", id, "thumbs", fname)
		if _, err := os.Stat(diskPath); err == nil {
			http.ServeFile(rw, req, diskPath)
			return
		}
		rw.Header().Set("Content-Type", "image/png")
		if err := db.StreamPageImage(req.Context(), id, "thumbs/"+fname, rw); err != nil {
			http.NotFound(rw, req)
		}
	})

	// Auth routes (public)
	r.Get("/login", a.LoginPage)
	r.Post("/login", a.LoginSubmit)
	r.Post("/logout", a.LogoutHandler)

	// Admin routes (protected)
	r.Group(func(r chi.Router) {
		r.Use(a.RequireAuth)
		r.Get("/admin", adminH.Index)
		r.Get("/admin/upload", adminH.UploadForm)
		r.Post("/admin/upload", adminH.Upload)
		r.Post("/admin/import", adminH.ImportURL)
		r.Get("/admin/flipbooks/{id}", adminH.Detail)
		r.Post("/admin/flipbooks/{id}/delete", adminH.Delete)
		r.Post("/admin/flipbooks/{id}/settings", adminH.Settings)
	})

	// API routes (protected by API key when set)
	r.Group(func(r chi.Router) {
		r.Use(APIAuth(cfg.APIKey))
		r.Get("/api/flipbooks", apiH.ListFlipbooks)
		r.Post("/api/flipbooks", apiH.UploadFlipbook)
		r.Post("/api/flipbooks/import", apiH.ImportURL)
		r.Get("/api/flipbooks/{id}", apiH.GetFlipbook)
		r.Delete("/api/flipbooks/{id}", apiH.DeleteFlipbook)
	})

	// Status endpoint (always accessible)
	r.Get("/api/flipbooks/{id}/status", apiH.FlipbookStatus)

	// Public viewers
	r.Get("/v/{slug}", viewerH.View)
	r.Get("/embed/{slug}", embedH.Embed)

	// Root redirect
	r.Get("/", func(rw http.ResponseWriter, req *http.Request) {
		http.Redirect(rw, req, "/admin", http.StatusFound)
	})

	return r
}

// SecurityHeaders adds standard security headers to all responses.
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "SAMEORIGIN")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		next.ServeHTTP(w, r)
	})
}

// APIAuth middleware protects API routes with a bearer token.
func APIAuth(apiKey string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := r.Header.Get("Authorization")
			if token == "Bearer "+apiKey {
				next.ServeHTTP(w, r)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":"unauthorized"}`))
		})
	}
}
