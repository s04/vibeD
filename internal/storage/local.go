package storage

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

// LocalStorage stores artifact source code and manifests on the local filesystem.
type LocalStorage struct {
	basePath string
}

// NewLocalStorage creates a LocalStorage rooted at basePath.
func NewLocalStorage(basePath string) (*LocalStorage, error) {
	if err := os.MkdirAll(basePath, 0o755); err != nil {
		return nil, fmt.Errorf("creating storage base path %q: %w", basePath, err)
	}
	return &LocalStorage{basePath: basePath}, nil
}

func (s *LocalStorage) StoreSource(_ context.Context, artifactID string, files map[string]string) (*StorageRef, error) {
	srcDir := filepath.Join(s.basePath, artifactID, "src")
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		return nil, fmt.Errorf("creating source dir: %w", err)
	}

	for relPath, content := range files {
		fullPath := filepath.Join(srcDir, relPath)
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("creating dir for %q: %w", relPath, err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
			return nil, fmt.Errorf("writing file %q: %w", relPath, err)
		}
	}

	return &StorageRef{
		Backend:   "local",
		LocalPath: srcDir,
	}, nil
}

func (s *LocalStorage) StoreManifest(_ context.Context, artifactID string, manifests map[string][]byte) error {
	manifestDir := filepath.Join(s.basePath, artifactID, "manifests")
	if err := os.MkdirAll(manifestDir, 0o755); err != nil {
		return fmt.Errorf("creating manifest dir: %w", err)
	}

	for filename, content := range manifests {
		fullPath := filepath.Join(manifestDir, filename)
		if err := os.WriteFile(fullPath, content, 0o644); err != nil {
			return fmt.Errorf("writing manifest %q: %w", filename, err)
		}
	}
	return nil
}

func (s *LocalStorage) GetSourcePath(_ context.Context, artifactID string) (string, error) {
	srcDir := filepath.Join(s.basePath, artifactID, "src")
	if _, err := os.Stat(srcDir); err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("source not found for artifact %q", artifactID)
		}
		return "", err
	}
	return srcDir, nil
}

func (s *LocalStorage) Delete(_ context.Context, artifactID string) error {
	artifactDir := filepath.Join(s.basePath, artifactID)
	if err := os.RemoveAll(artifactDir); err != nil {
		return fmt.Errorf("deleting artifact dir: %w", err)
	}
	return nil
}
