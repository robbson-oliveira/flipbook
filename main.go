package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/jonradoff/flipbook/internal/config"
	"github.com/jonradoff/flipbook/internal/converter"
	"github.com/jonradoff/flipbook/internal/database"
	"github.com/jonradoff/flipbook/internal/mcp"
	"github.com/jonradoff/flipbook/internal/server"
	"github.com/jonradoff/flipbook/internal/storage"
	"github.com/jonradoff/flipbook/internal/worker"
	"golang.org/x/crypto/bcrypt"
	"net/http"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "set-password":
			runSetPassword()
			return
		case "backfill-gridfs":
			runBackfillGridFS()
			return
		case "mcp":
			mcp.Run()
			return
		case "help":
			printHelp()
			return
		default:
			fmt.Fprintf(os.Stderr, "Unknown command: %s\n", os.Args[1])
			printHelp()
			os.Exit(1)
		}
	}
	runServer()
}

func printHelp() {
	fmt.Println("Usage: flipbook [command]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  (no command)       Start the web server")
	fmt.Println("  set-password       Set the admin password")
	fmt.Println("  backfill-gridfs    Upload existing originals to GridFS backup")
	fmt.Println("  mcp                Start the MCP server (stdin/stdout)")
	fmt.Println("  help               Show this help message")
}

func runSetPassword() {
	cfg := config.Load()
	db, err := database.Open(context.Background(), cfg.MongoURI, cfg.MongoDB)
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer db.Close(context.Background())

	fmt.Print("Enter new admin password: ")
	var password string
	fmt.Scanln(&password)
	if len(password) < 8 {
		fmt.Println("Error: Password must be at least 8 characters.")
		os.Exit(1)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		log.Fatalf("Failed to hash password: %v", err)
	}
	if err := db.SetSetting("admin_password_hash", string(hash)); err != nil {
		log.Fatalf("Failed to save password: %v", err)
	}
	fmt.Println("Admin password set successfully.")
}

func runBackfillGridFS() {
	cfg := config.Load()
	db, err := database.Open(context.Background(), cfg.MongoURI, cfg.MongoDB)
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer db.Close(context.Background())

	store := storage.New(filepath.Join(cfg.DataDir, "flipbooks"))
	flipbooks, err := db.ListFlipbooks()
	if err != nil {
		log.Fatalf("Failed to list flipbooks: %v", err)
	}

	for _, fb := range flipbooks {
		if fb.GridFSFileID != "" {
			fmt.Printf("  SKIP %s (%s) — already backed up\n", fb.ID, fb.Title)
			continue
		}
		ext := filepath.Ext(fb.Filename)
		srcPath := store.OriginalPath(fb.ID, ext)
		f, err := os.Open(srcPath)
		if err != nil {
			fmt.Printf("  MISS %s (%s) — original not found: %s\n", fb.ID, fb.Title, srcPath)
			continue
		}
		gridfsID, err := db.UploadToGridFS(context.Background(), fb.Filename, f)
		f.Close()
		if err != nil {
			fmt.Printf("  FAIL %s (%s) — GridFS upload: %v\n", fb.ID, fb.Title, err)
			continue
		}
		if err := db.SetGridFSFileID(fb.ID, gridfsID); err != nil {
			fmt.Printf("  FAIL %s (%s) — save ID: %v\n", fb.ID, fb.Title, err)
			continue
		}
		fmt.Printf("  OK   %s (%s) — backed up as %s\n", fb.ID, fb.Title, gridfsID)
	}
	fmt.Println("Backfill complete.")
}

func runServer() {
	cfg := config.Load()
	os.MkdirAll(cfg.DataDir, 0755)

	db, err := database.Open(context.Background(), cfg.MongoURI, cfg.MongoDB)
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer db.Close(context.Background())

	store := storage.New(filepath.Join(cfg.DataDir, "flipbooks"))
	conv := converter.New(cfg.LibreOfficeBin, filepath.Join(cfg.DataDir, "tmp"), cfg.ConversionDPI, cfg.ThumbnailDPI)

	w := worker.New(db, store, conv)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	w.Start(ctx)

	// Re-queue stuck jobs
	for _, status := range []string{"converting", "regenerating"} {
		stuck, _ := db.GetFlipbooksByStatus(status)
		for _, fb := range stuck {
			log.Printf("Re-queuing stuck %s: %s", status, fb.ID)
			db.UpdateStatus(fb.ID, "pending", "")
			ext := filepath.Ext(fb.Filename)
			w.Enqueue(worker.Job{FlipbookID: fb.ID, SourcePath: store.OriginalPath(fb.ID, ext)})
		}
	}

	// Integrity check: restore missing pages from GridFS
	go func() {
		readyFlipbooks, err := db.GetFlipbooksByStatus("ready")
		if err != nil {
			log.Printf("Integrity check: failed to query flipbooks: %v", err)
			return
		}
		for _, fb := range readyFlipbooks {
			if store.HasPages(fb.ID) {
				continue
			}
			if fb.GridFSFileID == "" {
				log.Printf("Integrity check: %s (%s) missing pages but no GridFS backup, skipping", fb.ID, fb.Title)
				continue
			}
			log.Printf("Integrity check: %s (%s) missing pages, restoring from GridFS", fb.ID, fb.Title)
			ext := filepath.Ext(fb.Filename)
			dstPath := store.OriginalPath(fb.ID, ext)
			if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
				log.Printf("Integrity check: failed to create dir for %s: %v", fb.ID, err)
				continue
			}
			f, err := os.Create(dstPath)
			if err != nil {
				log.Printf("Integrity check: failed to create file for %s: %v", fb.ID, err)
				continue
			}
			_, err = db.DownloadFromGridFS(context.Background(), fb.GridFSFileID, f)
			f.Close()
			if err != nil {
				log.Printf("Integrity check: GridFS download failed for %s: %v", fb.ID, err)
				os.Remove(dstPath)
				db.UpdateStatus(fb.ID, "error", "Failed to restore from backup: "+err.Error())
				continue
			}
			db.UpdateStatus(fb.ID, "regenerating", "")
			w.Enqueue(worker.Job{FlipbookID: fb.ID, SourcePath: dstPath})
		}
		log.Println("Integrity check complete")
	}()

	tmpl := server.ParseTemplates()
	h := server.BuildHandler(cfg, db, store, w, tmpl)

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           h,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      120 * time.Second,
		IdleTimeout:       120 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	log.Printf("Flipbook server starting on :%s", cfg.Port)
	log.Printf("Admin UI:  %s/admin", cfg.BaseURL)
	log.Printf("API key:   %s", cfg.APIKey)
	log.Printf("LibreOffice: %s", cfg.LibreOfficeBin)

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		log.Println("Shutting down...")
		cancel()
		srv.Shutdown(context.Background())
	}()

	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
