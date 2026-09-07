// Package appdir resolves where the application keeps its data.
package appdir

import (
	"os"
	"path/filepath"
)

const dirName = "proces-verbal-transare"

// DBPath returns the SQLite file path inside the per-user config directory,
// creating the directory if it does not exist.
func DBPath() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(base, dirName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return filepath.Join(dir, "data.db"), nil
}
