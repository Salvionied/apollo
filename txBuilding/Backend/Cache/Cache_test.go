package Cache

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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
	var data string
	found := Get("nonexistent_key_12345", &data)
	if found {
		t.Error("Get returned true for nonexistent key, expected false")
	}
}

func TestPathTraversalPrevention(t *testing.T) {
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

func TestCacheUsesSystemTempDir(t *testing.T) {
	dir := getCacheDir()
	tempDir := os.TempDir()

	// Check that cache dir starts with temp dir
	if !strings.HasPrefix(dir, tempDir) {
		t.Errorf("Cache directory %q is not under system temp dir %q", dir, tempDir)
	}

	if filepath.Base(dir) != "apollo-cache" {
		t.Errorf("Cache directory should be named 'apollo-cache', got %q", filepath.Base(dir))
	}
}
