package router

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestPathTraversalVulnerability demonstrates how an attacker can escape
// the working directory using path traversal sequences in the filename
func TestPathTraversalVulnerability(t *testing.T) {
	// Create a temporary working directory
	workingDir := t.TempDir()
	cfg := &RouterRuntimeConfig{
		TempStringLength: 4,
		WorkingDir:       workingDir,
		MaxUploadSize:    32 << 20,
	}

	// Create a test file outside the working directory
	outsideDir := filepath.Dir(workingDir)
	outsideFile := filepath.Join(outsideDir, "sensitive_file.txt")
	if err := os.WriteFile(outsideFile, []byte("sensitive data"), 0644); err != nil {
		t.Fatalf("failed to create outside file: %v", err)
	}
	defer os.Remove(outsideFile)

	// Create a multipart request with a path traversal payload
	payload := new(bytes.Buffer)
	writer := multipart.NewWriter(payload)

	// This filename attempts to write outside the working directory
	maliciousFilename := "../sensitive_file.txt"
	part, err := writer.CreateFormFile("file", maliciousFilename)
	if err != nil {
		t.Fatalf("failed to create form file: %v", err)
	}

	// Write some content that would overwrite the sensitive file
	if _, err := io.WriteString(part, "attacker controlled content"); err != nil {
		t.Fatalf("failed to write to part: %v", err)
	}
	writer.Close()

	// Create the HTTP request
	req := httptest.NewRequest("POST", "/", payload)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Call handleFileUpload
	filename, err := handleFileUpload(req, cfg)
	if err != nil {
		// Currently the code doesn't prevent this, but let's see what path it returns
		t.Logf("Error (expected if fixed): %v", err)
	}

	t.Logf("Returned filename: %s", filename)

	// Check if the file was written outside the working directory
	destFile := filepath.Join(cfg.WorkingDir, filename)
	t.Logf("Intended destination: %s", destFile)

	// The vulnerability: the actual file might be written to:
	actualPath := filepath.Join(cfg.WorkingDir, "../sensitive_file.txt")
	actualPath = filepath.Clean(actualPath) // Resolve .. sequences
	t.Logf("Vulnerable path created: %s", actualPath)

	// Check if sensitive_file.txt was overwritten
	content, err := os.ReadFile(outsideFile)
	if err == nil {
		if bytes.Contains(content, []byte("attacker controlled content")) {
			t.Error("VULNERABILITY CONFIRMED: Attacker was able to write outside working directory!")
		}
	}
}

func TestSimpleFormStore_Save(t *testing.T) {
	dest := t.TempDir()
	store := &SimpleFormStore{
		File:     "test_simple.txt",
		Contents: "hello world simple",
	}

	filename, written, err := store.Save(dest)
	if err != nil {
		t.Fatalf("unexpected error saving simple form: %v", err)
	}

	if written != len("hello world simple") {
		t.Errorf("expected %d bytes written, got %d", len("hello world simple"), written)
	}

	content, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("failed to read written file: %v", err)
	}
	if string(content) != "hello world simple" {
		t.Errorf("expected 'hello world simple', got '%s'", string(content))
	}
}

func TestHandleFileUpload_SimpleForm(t *testing.T) {
	workingDir := t.TempDir()
	cfg := &RouterRuntimeConfig{
		TempStringLength: 4,
		WorkingDir:       workingDir,
		MaxUploadSize:    1024,
	}

	form := url.Values{}
	form.Add("filename", "orchestrated_simple.txt")
	form.Add("contents", "orchestrated contents")

	req := httptest.NewRequest("POST", "/", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	savedPath, err := handleFileUpload(req, cfg)
	if err != nil {
		t.Fatalf("unexpected error in handleFileUpload: %v", err)
	}

	fullPath := filepath.Join(workingDir, savedPath)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		t.Fatalf("failed to read saved file: %v", err)
	}
	if string(content) != "orchestrated contents" {
		t.Errorf("expected 'orchestrated contents', got '%s'", string(content))
	}
}

func init() {
	initStoreFactories()
}

func TestHandleFileUpload_MultipartForm(t *testing.T) {
	workingDir := t.TempDir()
	cfg := &RouterRuntimeConfig{
		TempStringLength: 4,
		WorkingDir:       workingDir,
		MaxUploadSize:    1024,
	}

	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "orchestrated_multipart.txt")
	if err != nil {
		t.Fatalf("failed to create form file: %v", err)
	}
	part.Write([]byte("multipart contents"))
	writer.Close()

	req := httptest.NewRequest("POST", "/", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	savedPath, err := handleFileUpload(req, cfg)
	if err != nil {
		t.Fatalf("unexpected error in handleFileUpload: %v", err)
	}

	fullPath := filepath.Join(workingDir, savedPath)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		t.Fatalf("failed to read saved file: %v", err)
	}
	if string(content) != "multipart contents" {
		t.Errorf("expected 'multipart contents', got '%s'", string(content))
	}
}

func TestUploadSizeLimit(t *testing.T) {
	workingDir := t.TempDir()
	cfg := &RouterRuntimeConfig{
		WorkingDir:       workingDir,
		TempStringLength: 4,
		MaxUploadSize:    512, // 512 bytes limit
	}

	server := NewRouter(cfg.MaxUploadSize, cfg.TempStringLength, cfg.WorkingDir)
	t.Run("Exceeds Limit", func(t *testing.T) {
		body := new(bytes.Buffer)
		writer := multipart.NewWriter(body)
		part, err := writer.CreateFormFile("file", "large.txt")
		if err != nil {
			t.Fatalf("failed to create form file: %v", err)
		}
		part.Write(bytes.Repeat([]byte("a"), 1024)) // 1KB
		writer.Close()

		req := httptest.NewRequest("POST", "/", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		rr := httptest.NewRecorder()

		server.UploadFileHandler(cfg)(rr, req)

		if rr.Code == http.StatusOK {
			t.Errorf("Expected failure for file exceeding size limit, but got StatusOK")
		}
	})

	t.Run("Within Limit", func(t *testing.T) {
		body := new(bytes.Buffer)
		writer := multipart.NewWriter(body)
		part, err := writer.CreateFormFile("file", "small.txt")
		if err != nil {
			t.Fatalf("failed to create form file: %v", err)
		}
		part.Write(bytes.Repeat([]byte("a"), 256)) // 256B
		writer.Close()

		req := httptest.NewRequest("POST", "/", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		rr := httptest.NewRecorder()

		server.UploadFileHandler(cfg)(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected success for file within size limit, but got %v", rr.Code)
		}
	})
}
