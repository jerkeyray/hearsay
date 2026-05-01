package game

import (
	"errors"
	"os"
	"path/filepath"
)

// SaveDir returns the directory where session SQLite files live.
// Honors HEARSAY_HOME, falls back to $HOME/.hearsay.
func SaveDir() (string, error) {
	if h := os.Getenv("HEARSAY_HOME"); h != "" {
		return filepath.Join(h, "saves"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	if home == "" {
		return "", errors.New("game: no home directory")
	}
	return filepath.Join(home, ".hearsay", "saves"), nil
}

// EnsureSaveDir creates the save directory if it does not exist and
// returns its absolute path.
func EnsureSaveDir() (string, error) {
	d, err := SaveDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(d, 0o755); err != nil {
		return "", err
	}
	return d, nil
}

// SavePath is the per-session SQLite path: <saveDir>/<caseID>-<runID>.db.
func SavePath(saveDir, caseID, runID string) string {
	return filepath.Join(saveDir, caseID+"-"+runID+".db")
}
