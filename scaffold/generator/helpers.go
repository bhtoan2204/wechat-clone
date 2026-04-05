package generator

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func structExistsInDir(dir, structName string) bool {
	found := false
	_ = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || found || d == nil || d.IsDir() || filepath.Ext(path) != ".go" {
			return nil
		}
		content, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}
		if strings.Contains(string(content), "type "+structName+" struct") {
			found = true
		}
		return nil
	})
	return found
}
