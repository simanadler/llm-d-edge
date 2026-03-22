package downloader

import (
	"context"
	"io"

	"github.com/simanadler/llm-d-edge/model-manager/pkg/types"
)

// ModelDownloader is an interface for downloading models from various sources
type ModelDownloader interface {
	// Download downloads a model to the specified destination
	Download(ctx context.Context, model types.ModelMetadata, dest string, opts DownloadOptions) error
	
	// GetDownloadURL returns the download URL for a model
	GetDownloadURL(model types.ModelMetadata, opts DownloadOptions) (string, error)
	
	// SupportsModel checks if this downloader supports the given model
	SupportsModel(model types.ModelMetadata) bool
	
	// Name returns the downloader name
	Name() string
}

// DownloadOptions contains options for downloading a model
type DownloadOptions struct {
	// Format specifies the model format (mlx, gguf, safetensors, etc.)
	Format string
	
	// Quantization specifies the quantization level (q4, q8, fp16, etc.)
	Quantization string
	
	// ProgressCallback is called with download progress updates
	ProgressCallback func(progress DownloadProgress)
	
	// Resume allows resuming interrupted downloads
	Resume bool
	
	// Verify enables checksum verification
	Verify bool
}

// DownloadProgress represents download progress information
type DownloadProgress struct {
	// BytesDownloaded is the number of bytes downloaded so far
	BytesDownloaded int64
	
	// TotalBytes is the total size of the download
	TotalBytes int64
	
	// Percentage is the download percentage (0-100)
	Percentage float64
	
	// Speed is the current download speed in bytes/sec
	Speed int64
	
	// ETA is the estimated time remaining in seconds
	ETA int64
	
	// Status is the current status message
	Status string
}

// ProgressWriter wraps an io.Writer to track download progress
type ProgressWriter struct {
	writer   io.Writer
	total    int64
	written  int64
	callback func(DownloadProgress)
}

// NewProgressWriter creates a new progress tracking writer
func NewProgressWriter(w io.Writer, total int64, callback func(DownloadProgress)) *ProgressWriter {
	return &ProgressWriter{
		writer:   w,
		total:    total,
		callback: callback,
	}
}

// Write implements io.Writer
func (pw *ProgressWriter) Write(p []byte) (int, error) {
	n, err := pw.writer.Write(p)
	pw.written += int64(n)
	
	if pw.callback != nil {
		percentage := 0.0
		if pw.total > 0 {
			percentage = float64(pw.written) / float64(pw.total) * 100
		}
		
		pw.callback(DownloadProgress{
			BytesDownloaded: pw.written,
			TotalBytes:      pw.total,
			Percentage:      percentage,
			Status:          "downloading",
		})
	}
	
	return n, err
}

// Made with Bob