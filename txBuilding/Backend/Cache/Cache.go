package Cache

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

var (
	cacheDir     string
	cacheDirOnce sync.Once
)

// getCacheDir returns the cache directory path. Directory creation and
// permission checks are handled by ensureCacheDir.
func getCacheDir() string {
	cacheDirOnce.Do(func() {
		if cacheDir != "" {
			return
		}
		userCacheDir, err := os.UserCacheDir()
		if err == nil && userCacheDir != "" {
			cacheDir = filepath.Join(userCacheDir, "apollo")
			return
		}
		cacheDir = filepath.Join(
			os.TempDir(),
			"apollo-cache-"+strconv.Itoa(os.Getuid()),
		)
	})
	return cacheDir
}

func ensureCacheDir() error {
	dir := getCacheDir()
	info, err := os.Lstat(dir)
	if errors.Is(err, os.ErrNotExist) {
		if err := os.MkdirAll(dir, 0700); err != nil {
			return fmt.Errorf("create cache directory: %w", err)
		}
		info, err = os.Lstat(dir)
	}
	if err != nil {
		return fmt.Errorf("stat cache directory: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("cache directory %q must not be a symlink", dir)
	}
	if !info.IsDir() {
		return fmt.Errorf("cache path %q is not a directory", dir)
	}
	if info.Mode().Perm()&0077 != 0 {
		if err := os.Chmod(dir, 0700); err != nil {
			return fmt.Errorf("secure cache directory permissions: %w", err)
		}
	}
	return nil
}

// sanitizeKey removes path traversal attempts and invalid characters
// from cache keys.
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
	if err := ensureCacheDir(); err != nil {
		return false
	}
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
	if err := ensureCacheDir(); err != nil {
		return fmt.Errorf("cache set: %w", err)
	}
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
