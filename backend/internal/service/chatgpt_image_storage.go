package service

import (
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ChatGPTImageStorage handles local storage of ChatGPT-generated images.
type ChatGPTImageStorage struct {
	baseDir    string // 数据目录，如 /app/data/images
	publicBase string // 公网 URL 前缀，如 http://example.com:26019
}

// NewChatGPTImageStorage creates a new image storage service.
// dataDir is the base data directory (e.g. "./data"); images are stored under {dataDir}/images/.
// publicBaseURL is the public-facing base URL (e.g. "http://example.com:26019").
func NewChatGPTImageStorage(dataDir string, publicBaseURL string) *ChatGPTImageStorage {
	return &ChatGPTImageStorage{
		baseDir:    filepath.Join(dataDir, "images"),
		publicBase: strings.TrimRight(publicBaseURL, "/"),
	}
}

// Save stores image data to a local file and returns the relative path and public URL.
// The relative path follows the pattern: YYYY/MM/DD/{timestamp}_{short_uuid}.{ext}
// ext defaults to "png" when empty.
func (s *ChatGPTImageStorage) Save(imageData []byte, ext string) (relativePath string, publicURL string, err error) {
	if ext == "" {
		ext = "png"
	}
	ext = strings.TrimPrefix(ext, ".")

	// Generate short UUID (first 8 hex chars)
	uuidBytes := make([]byte, 16)
	if _, err := rand.Read(uuidBytes); err != nil {
		return "", "", fmt.Errorf("generate uuid: %w", err)
	}
	shortUUID := fmt.Sprintf("%x", uuidBytes)[:8]

	// Build directory structure: YYYY/MM/DD
	now := time.Now()
	dateDir := filepath.Join(
		fmt.Sprintf("%04d", now.Year()),
		fmt.Sprintf("%02d", now.Month()),
		fmt.Sprintf("%02d", now.Day()),
	)

	// Build filename: {unix_timestamp}_{short_uuid}.{ext}
	filename := fmt.Sprintf("%d_%s.%s", now.Unix(), shortUUID, ext)

	// Full relative path (within images directory)
	relativePath = filepath.ToSlash(filepath.Join(dateDir, filename))

	// Ensure date directory exists
	fullDir := filepath.Join(s.baseDir, dateDir)
	if err := os.MkdirAll(fullDir, 0755); err != nil {
		return "", "", fmt.Errorf("create directory: %w", err)
	}

	// Write file
	fullPath := filepath.Join(s.baseDir, relativePath)
	if err := os.WriteFile(fullPath, imageData, 0644); err != nil {
		return "", "", fmt.Errorf("write file: %w", err)
	}

	publicURL = fmt.Sprintf("%s/images/%s", s.publicBase, relativePath)
	return relativePath, publicURL, nil
}

// GetFullPath returns the full filesystem path for a given relative image path.
func (s *ChatGPTImageStorage) GetFullPath(relativePath string) string {
	return filepath.Join(s.baseDir, filepath.FromSlash(relativePath))
}

// GetURL returns the public URL for a given relative image path.
func (s *ChatGPTImageStorage) GetURL(relativePath string) string {
	return fmt.Sprintf("%s/images/%s", s.publicBase, strings.TrimLeft(filepath.ToSlash(relativePath), "/"))
}
