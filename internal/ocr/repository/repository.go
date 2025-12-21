package repository

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Repository implements the ocr.Repository interface for file operations
type Repository struct {
	baseDir    string
	outputPath string
}

// New creates a new Repository instance with the specified base directory and output path.
// If baseDir is empty, it defaults to the current working directory.
// If outputPath is relative, it will be joined with baseDir.
func New(baseDir, outputPath string) (*Repository, error) {
	if baseDir == "" {
		wd, _ := os.Getwd()
		baseDir = wd
	}

	// If output path is relative, join it with the base directory
	if !filepath.IsAbs(outputPath) {
		outputPath = filepath.Join(baseDir, outputPath)
	}

	// Check if image directory exists
	if info, err := os.Stat(baseDir); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDirectoryNotFound, err)
	} else if !info.IsDir() {
		return nil, fmt.Errorf("%w: path is not a directory", ErrDirectoryNotFound)
	}

	return &Repository{
		baseDir:    baseDir,
		outputPath: outputPath,
	}, nil
}

var (
	// ErrDirectoryNotFound is returned when the specified directory does not exist
	ErrDirectoryNotFound = fmt.Errorf("directory not found")
	// ErrImageNotFound is returned when the specified image file does not exist
	ErrImageNotFound = fmt.Errorf("image not found")
	// ErrFailedToSave is returned when saving output fails
	ErrFailedToSave = fmt.Errorf("failed to save output")
)

// GetImageNames returns sorted image filenames from the repository's base directory.
func (r *Repository) GetImageNames() ([]string, error) {
	var imageNames []string
	imageExts := map[string]bool{
		".jpg":  true,
		".jpeg": true,
		".png":  true,
		".gif":  true,
		".bmp":  true,
		".webp": true,
	}

	err := filepath.WalkDir(r.baseDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if imageExts[ext] {
			imageNames = append(imageNames, filepath.Base(path))
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk directory: %w", err)
	}

	// Sort alphabetically
	sort.Strings(imageNames)

	return imageNames, nil
}

// LoadImageByName loads image data by filename from the repository's base directory.
func (r *Repository) LoadImageByName(filename string) ([]byte, error) {
	path := filepath.Join(r.baseDir, filename)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%w: %s", ErrImageNotFound, filename)
		}
		return nil, fmt.Errorf("%w: %v", ErrImageNotFound, err)
	}
	return data, nil
}

// SaveOutput saves the output text to the repository's configured output path
func (r *Repository) SaveOutput(content string) error {
	err := os.WriteFile(r.outputPath, []byte(content), 0644)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrFailedToSave, err)
	}
	return nil
}
