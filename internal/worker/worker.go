package worker

import (
	"context"
	"log"
	"os"
	"path/filepath"

	"github.com/jonradoff/flipbook/internal/converter"
	"github.com/jonradoff/flipbook/internal/database"
	"github.com/jonradoff/flipbook/internal/models"
	"github.com/jonradoff/flipbook/internal/storage"
)

type Job struct {
	FlipbookID string
	SourcePath string
}

type Worker struct {
	jobs    chan Job
	db      *database.DB
	storage *storage.Storage
	conv    *converter.Converter
}

func New(db *database.DB, store *storage.Storage, conv *converter.Converter) *Worker {
	return &Worker{
		jobs:    make(chan Job, 100),
		db:      db,
		storage: store,
		conv:    conv,
	}
}

func (w *Worker) Start(ctx context.Context) {
	go func() {
		for {
			select {
			case job := <-w.jobs:
				w.processJob(ctx, job)
			case <-ctx.Done():
				return
			}
		}
	}()
}

func (w *Worker) Enqueue(job Job) {
	w.jobs <- job
}

func (w *Worker) processJob(ctx context.Context, job Job) {
	log.Printf("[worker] Starting conversion for %s", job.FlipbookID)

	if err := w.db.UpdateStatus(job.FlipbookID, models.StatusConverting, ""); err != nil {
		log.Printf("[worker] Failed to update status: %v", err)
		return
	}

	if err := w.storage.EnsureDirs(job.FlipbookID); err != nil {
		log.Printf("[worker] Failed to create dirs: %v", err)
		w.db.UpdateStatus(job.FlipbookID, models.StatusError, err.Error())
		return
	}

	result, err := w.conv.Convert(
		ctx,
		job.SourcePath,
		w.storage.PagesDir(job.FlipbookID),
		w.storage.ThumbsDir(job.FlipbookID),
	)
	if err != nil {
		log.Printf("[worker] Conversion failed for %s: %v", job.FlipbookID, err)
		w.db.UpdateStatus(job.FlipbookID, models.StatusError, err.Error())
		return
	}

	if err := w.db.UpdateConversionResult(job.FlipbookID, result.PageCount, result.Width, result.Height); err != nil {
		log.Printf("[worker] Failed to save result: %v", err)
		w.db.UpdateStatus(job.FlipbookID, models.StatusError, err.Error())
		return
	}

	log.Printf("[worker] Conversion complete for %s: %d pages", job.FlipbookID, result.PageCount)

	// Upload page images to GridFS so they can be served from serverless environments.
	go w.uploadPagesToGridFS(ctx, job.FlipbookID)
}

// uploadPagesToGridFS uploads all page and thumb PNGs to GridFS after conversion.
// This runs in a goroutine so it doesn't block the worker. Errors are logged but non-fatal.
func (w *Worker) uploadPagesToGridFS(ctx context.Context, flipbookID string) {
	for _, dir := range []string{w.storage.PagesDir(flipbookID), w.storage.ThumbsDir(flipbookID)} {
		entries, err := filepath.Glob(filepath.Join(dir, "*.png"))
		if err != nil || len(entries) == 0 {
			continue
		}
		for _, path := range entries {
			f, err := os.Open(path)
			if err != nil {
				log.Printf("[worker] GridFS upload: open %s: %v", path, err)
				continue
			}
			filename := filepath.Base(path)
			// Distinguish pages vs thumbs in the key
			if filepath.Base(dir) == "thumbs" {
				filename = "thumbs/" + filename
			} else {
				filename = "pages/" + filename
			}
			if err := w.db.UploadPageImage(ctx, flipbookID, filename, f); err != nil {
				log.Printf("[worker] GridFS upload: %s/%s: %v", flipbookID, filename, err)
			}
			f.Close()
		}
	}
	log.Printf("[worker] GridFS page upload complete for %s", flipbookID)
}
