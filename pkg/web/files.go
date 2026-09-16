package web

import (
	"os"
	"path/filepath"
)

func readPublicFile(dir, file string) ([]byte, error) {
	root, err := os.OpenRoot(dir)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	rel, err := filepath.Rel(dir, file)
	if err != nil {
		return nil, err
	}
	return root.ReadFile(rel)
}
