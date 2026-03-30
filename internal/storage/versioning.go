package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

const maxVersions = 10

// saveVersion copies the current file to .versions/ before overwriting.
// Does nothing if the file doesn't exist yet.
func saveVersion(path string) error {
	// Check if file exists
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // nothing to version
		}
		return fmt.Errorf("stat for versioning: %w", err)
	}
	if info.Size() == 0 {
		return nil // empty file, skip
	}

	dir := filepath.Dir(path)
	base := filepath.Base(path)
	versDir := filepath.Join(dir, ".versions")

	if err := os.MkdirAll(versDir, 0o700); err != nil {
		return fmt.Errorf("create versions dir: %w", err)
	}

	// Read current file
	data, err := os.ReadFile(filepath.Clean(path)) // #nosec G304
	if err != nil {
		return fmt.Errorf("read for versioning: %w", err)
	}

	// Write versioned copy
	ts := time.Now().Unix()
	versFile := filepath.Clean(filepath.Join(versDir, fmt.Sprintf("%s.v%d", base, ts)))
	if err := os.WriteFile(versFile, data, 0o600); err != nil { // #nosec G306 G703 -- versFile is built from filepath.Join(dataDir, ".versions", base+timestamp), no user input in filename components
		return fmt.Errorf("write version: %w", err)
	}

	// Prune old versions
	return pruneVersions(versDir, base)
}

// pruneVersions keeps only the newest maxVersions files for a given base name.
func pruneVersions(versDir, base string) error {
	entries, err := os.ReadDir(versDir)
	if err != nil {
		return nil // non-fatal
	}

	prefix := base + ".v"
	var versions []string
	for _, e := range entries {
		if !e.IsDir() && len(e.Name()) > len(prefix) && e.Name()[:len(prefix)] == prefix {
			versions = append(versions, e.Name())
		}
	}

	if len(versions) <= maxVersions {
		return nil
	}

	// Sort lexicographically (timestamps sort correctly as strings)
	sort.Strings(versions)

	// Remove oldest
	toRemove := versions[:len(versions)-maxVersions]
	for _, name := range toRemove {
		os.Remove(filepath.Join(versDir, name))
	}

	return nil
}

// ListVersions returns version timestamps for a file, newest first.
func ListVersions(path string) ([]int64, error) {
	dir := filepath.Dir(path)
	base := filepath.Base(path)
	versDir := filepath.Join(dir, ".versions")

	entries, err := os.ReadDir(versDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read versions dir: %w", err)
	}

	prefix := base + ".v"
	var timestamps []int64
	for _, e := range entries {
		name := e.Name()
		if !e.IsDir() && len(name) > len(prefix) && name[:len(prefix)] == prefix {
			var ts int64
			if _, err := fmt.Sscanf(name[len(prefix):], "%d", &ts); err == nil {
				timestamps = append(timestamps, ts)
			}
		}
	}

	// Sort newest first
	sort.Slice(timestamps, func(i, j int) bool { return timestamps[i] > timestamps[j] })
	return timestamps, nil
}
