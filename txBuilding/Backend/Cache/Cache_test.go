package Cache

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func useTempCacheDir(t *testing.T) {
	t.Helper()

	cacheDir = filepath.Join(t.TempDir(), "apollo-cache")
	cacheDirOnce = sync.Once{}
	t.Cleanup(func() {
		cacheDir = ""
		cacheDirOnce = sync.Once{}
	})
}

func TestSanitizeKey(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple key", "test", "test"},
		{"path traversal dots", "../../../etc/passwd", "___etc_passwd"},
		{"path traversal with slashes", "foo/../bar", "foo__bar"},
		{"backslash traversal", "foo\\..\\bar", "foo__bar"},
		{"mixed traversal", "../foo/bar\\..\\baz", "_foo_bar__baz"},
		{"empty after sanitize", "..", "default"},
		{"just dots", "...", "default"},
		{"normal with underscore", "latest_epoch", "latest_epoch"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeKey(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizeKey(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestGetSet(t *testing.T) {
	useTempCacheDir(t)

	// Test basic get/set functionality
	type TestData struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	testData := TestData{Name: "test", Value: 42}

	// Set should succeed
	err := Set("test_key", testData)
	if err != nil {
		t.Errorf("Set failed: %v", err)
	}

	// Get should succeed and return the same data
	var retrieved TestData
	found := Get("test_key", &retrieved)
	if !found {
		t.Error("Get returned false, expected true")
	}
	if retrieved.Name != testData.Name || retrieved.Value != testData.Value {
		t.Errorf("Get returned %+v, want %+v", retrieved, testData)
	}

	// Clean up
	path := filepath.Join(getCacheDir(), "test_key.json")
	os.Remove(path)
}

func TestGetMissingKey(t *testing.T) {
	useTempCacheDir(t)

	var data string
	found := Get("nonexistent_key_12345", &data)
	if found {
		t.Error("Get returned true for nonexistent key, expected false")
	}
}

func TestPathTraversalPrevention(t *testing.T) {
	useTempCacheDir(t)

	// Attempt to write to a path outside the cache directory
	maliciousKey := "../../../tmp/malicious"

	err := Set(maliciousKey, "test")
	if err != nil {
		// Set rejected the malicious key - this is acceptable protection
		t.Logf("Set rejected malicious key with error: %v", err)
	}

	// Verify the malicious path was not created (the critical check)
	maliciousPath := "/tmp/malicious.json"
	if _, err := os.Stat(maliciousPath); err == nil {
		t.Fatal("Path traversal attack succeeded - malicious file was created")
		os.Remove(maliciousPath)
	}

	// If Set succeeded, verify the file was written to the cache directory
	sanitized := sanitizeKey(maliciousKey)
	expectedPath := filepath.Join(getCacheDir(), sanitized+".json")
	if _, statErr := os.Stat(expectedPath); statErr == nil {
		// Clean up the safely-written file
		os.Remove(expectedPath)
	}
}

func TestCacheUsesPrivateDir(t *testing.T) {
	useTempCacheDir(t)

	dir := getCacheDir()
	if err := ensureCacheDir(); err != nil {
		t.Fatalf("ensureCacheDir failed: %v", err)
	}

	if filepath.Base(dir) != "apollo-cache" {
		t.Errorf(
			"Cache directory should be named 'apollo-cache', got %q",
			filepath.Base(dir),
		)
	}
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("stat cache dir: %v", err)
	}
	if info.Mode().Perm() != 0700 {
		t.Errorf("cache directory permissions = %o, want 0700", info.Mode().Perm())
	}
}
