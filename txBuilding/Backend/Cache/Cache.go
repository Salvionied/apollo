package Cache

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var (
	cacheDir     string
	cacheDirOnce sync.Once
)

// getCacheDir returns the cache directory, creating it if necessary.
// Uses os.TempDir() for a consistent, system-appropriate location.
func getCacheDir() string {
	cacheDirOnce.Do(func() {
		cacheDir = filepath.Join(os.TempDir(), "apollo-cache")
		// Best effort directory creation - errors handled in Get/Set
		_ = os.MkdirAll(cacheDir, 0755)
	})
	return cacheDir
}

// sanitizeKey removes path traversal attempts and invalid characters from cache keys.
// Returns a safe filename that cannot escape the cache directory.
func sanitizeKey(key string) string {
	// Remove any path separators and parent directory references
	key = strings.ReplaceAll(key, "..", "")
	key = strings.ReplaceAll(key, "/", "_")
	key = strings.ReplaceAll(key, "\\", "_")
	// Use filepath.Base as final safety check
	key = filepath.Base(key)
	// Ensure non-empty key
	if key == "" || key == "." {
		key = "default"
	}
	return key
}

// Get retrieves a cached value by key. Returns true if the value was found
// and successfully unmarshaled, false otherwise.
func Get[T any](key string, val *T) bool {
	key = sanitizeKey(key)
	path := filepath.Join(getCacheDir(), key+".json")
	dat, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	err = json.Unmarshal(dat, val)
	return err == nil
}

// Set stores a value in the cache with the given key.
// Returns an error if marshaling or writing fails.
func Set[T any](key string, value T) error {
	key = sanitizeKey(key)
	val, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("cache set %q: marshal error: %w", key, err)
	}
	path := filepath.Join(getCacheDir(), key+".json")
	if err := os.WriteFile(path, val, 0600); err != nil {
		return fmt.Errorf("cache set %q: write error: %w", key, err)
	}
	return nil
}
