// Package handler is the Vercel serverless function entry point.
package handler

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"sync"
	"time"

	"github.com/jonradoff/flipbook/internal/config"
	"github.com/jonradoff/flipbook/internal/database"
	"github.com/jonradoff/flipbook/internal/server"
	"github.com/jonradoff/flipbook/internal/storage"
)

var (
	mu         sync.RWMutex
	appHandler http.Handler
)

// Handler is the Vercel Go serverless function entry point.
func Handler(w http.ResponseWriter, r *http.Request) {
	h := getHandler()
	h.ServeHTTP(w, r)
}

func getHandler() http.Handler {
	mu.RLock()
	if appHandler != nil {
		defer mu.RUnlock()
		return appHandler
	}
	mu.RUnlock()

	mu.Lock()
	defer mu.Unlock()

	// Double check
	if appHandler != nil {
		return appHandler
	}

	cfg := config.Load()
	if cfg.MongoURI == "" {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprint(w, `<!DOCTYPE html>
<html>
<head><title>Flipbook - Configuração Necessária</title><style>body{font-family:sans-serif;background:#0d0d1a;color:#fff;display:flex;align-items:center;justify-content:center;height:100vh;margin:0;padding:20px;text-align:center;line-height:1.6;} .box{max-width:550px;background:rgba(255,255,255,0.05);padding:30px;border-radius:12px;border:1px solid rgba(255,255,255,0.1);} h2{color:#6366f1;} code{background:#1e1e38;padding:3px 8px;border-radius:4px;color:#a5b4fc;}</style></head>
<body>
<div class="box">
  <h2>Flipbook Vercel</h2>
  <p>O servidor está ativo, mas aguardando a configuração do banco de dados.</p>
  <p>Configure a variável de ambiente <code>FLIPBOOK_MONGO_URI</code> no painel da Vercel (Project Settings &rarr; Environment Variables) com a sua connection string do MongoDB Atlas.</p>
</div>
</body></html>`)
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	db, err := database.Open(ctx, cfg.MongoURI, cfg.MongoDB)
	if err != nil {
		log.Printf("[vercel] MongoDB connection error: %v", err)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprintf(w, `<!DOCTYPE html>
<html>
<head><title>Flipbook - Erro de Conexão</title><style>body{font-family:sans-serif;background:#0d0d1a;color:#fff;display:flex;align-items:center;justify-content:center;height:100vh;margin:0;padding:20px;text-align:center;line-height:1.6;} .box{max-width:550px;background:rgba(255,255,255,0.05);padding:30px;border-radius:12px;border:1px solid rgba(255,255,255,0.1);} h2{color:#ef4444;} code{background:#1e1e38;padding:3px 8px;border-radius:4px;color:#fca5a5;}</style></head>
<body>
<div class="box">
  <h2>Erro de Conexão com MongoDB</h2>
  <p>Não foi possível conectar ao MongoDB Atlas. Verifique se o IP 0.0.0.0/0 está liberado no MongoDB Atlas Network Access e se as credenciais estão corretas.</p>
  <p><code>%v</code></p>
</div>
</body></html>`, err)
		})
	}

	store := storage.New(filepath.Join("/tmp", "flipbooks"))
	tmpl := server.ParseTemplates()

	appHandler = server.BuildHandler(cfg, db, store, nil, tmpl)
	return appHandler
}
