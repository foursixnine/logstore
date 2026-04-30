package router

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/foursixnine/logstore/internal/utils"
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

func TestE2EUpload(t *testing.T) {
	workingDir := t.TempDir()
	cfg := &RouterRuntimeConfig{
		WorkingDir:       workingDir,
		TempStringLength: 4,
		MaxUploadSize:    512, // 512 bytes limit
	}

	server := NewRouter(cfg.MaxUploadSize, cfg.TempStringLength, cfg.WorkingDir)
	t.Run("Exceeds Limit", func(t *testing.T) {
		req := setupUploadRequest(t, "large.txt", "/", 1024)
		rr := httptest.NewRecorder()

		server.UploadFileHandler(rr, req)

		if rr.Code == http.StatusOK {
			t.Errorf("Expected failure for file exceeding size limit, but got StatusOK")
		}
	})

	var session string

	t.Run("Within Limit", func(t *testing.T) {
		req := setupUploadRequest(t, "small.txt", "/", 256)
		rr := httptest.NewRecorder()

		server.UploadFileHandler(rr, req)
		result := rr.Result()

		if result.StatusCode != http.StatusOK {
			t.Errorf("Expected success for file within size limit, but got %v", rr.Code)
		}

		re, err := regexp.Compile(`example.com/logs/(.*)/small.txt`)
		if err != nil {
			t.Logf("Error found: %v", err)
			t.Fail()
		}

		matches := re.FindSubmatch(rr.Body.Bytes())
		if matches == nil {
			t.Fatalf("Path part not found in body: %s", matches)
		}

		t.Logf("result: %s", matches[1])
		session = string(matches[1])

	})

	t.Run("Download archive", func(t *testing.T) {
		sessionURL, err := url.JoinPath("/d", session, "tar")
		if err != nil {
			t.Fatalf("Found error %v", err)
		}

		req := httptest.NewRequest("GET", sessionURL, nil)
		rr := httptest.NewRecorder()

		req.SetPathValue("session", session)
		req.SetPathValue("type", "tar")

		server.ArchiveHandler(rr, req)
		result := rr.Result()

		if result.StatusCode != http.StatusOK {
			t.Errorf("Expected success for file within size limit, but got %v", rr.Code)
		}

		cd := result.Header.Get("Content-Disposition")
		expected := fmt.Sprintf("attachment; filename=\"%s-logs.%s\"", session, "tar")
		if cd != expected {
			t.Fatalf("File header is incorrect, wanted %s, got %s", expected, cd)
		}

	})

	t.Run("Upload to session", func(t *testing.T) {

		sessionURL, err := url.JoinPath("/", session)
		req := setupUploadRequest(t, "big.txt", sessionURL, 256)
		req.SetPathValue("session", session)

		rr := httptest.NewRecorder()

		server.UploadFileHandler(rr, req)

		result := rr.Result()
		if result.StatusCode != http.StatusOK {
			t.Errorf("Expected success for file within size limit, but got %v", rr.Code)
		}

		re, err := regexp.Compile(`example.com/logs/(.*)/big.txt`)
		if err != nil {
			t.Logf("Error found: %v", err)
			t.Fail()
		}

		matches := re.FindSubmatch(rr.Body.Bytes())
		if matches == nil {
			t.Fatalf("Path part not found in body: %s", matches)
		}

		t.Logf("result: %s", matches[1])
		if session != string(matches[1]) {
			t.Fatalf("Session error, wanted %s but got %s", session, matches[1])
		}

	})

}

func setupUploadRequest(t *testing.T, name string, path string, size int) *http.Request {
	t.Helper()
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	defer writer.Close()

	part, err := writer.CreateFormFile("file", name)
	if err != nil {
		t.Fatalf("failed to create form file: %v", err)
	}

	contents := utils.RandomString(1)
	part.Write(bytes.Repeat([]byte(contents), size)) // 256B

	req := httptest.NewRequest("POST", path, body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req
}
