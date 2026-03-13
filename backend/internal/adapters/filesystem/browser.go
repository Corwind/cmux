package filesystem

import (
	"fmt"
	"os"
	"sort"

	"github.com/Corwind/cmux/backend/internal/ports"
)

type Browser struct{}

func NewBrowser() *Browser {
	return &Browser{}
}

func (b *Browser) ListDir(path string, showHidden bool) ([]ports.DirEntry, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	var result []ports.DirEntry
	for _, entry := range entries {
		if !showHidden && entry.Name()[0] == '.' {
			continue
		}
		result = append(result, ports.DirEntry{
			Name:  entry.Name(),
			IsDir: entry.IsDir(),
		})
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].IsDir != result[j].IsDir {
			return result[i].IsDir
		}
		return result[i].Name < result[j].Name
	})

	return result, nil
}

func (b *Browser) HomeDir() (string, error) {
	return os.UserHomeDir()
}
