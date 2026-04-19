package logstore

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// TestPathTraversalVulnerability demonstrates how an attacker can escape
// the working directory using path traversal sequences in the filename
func TestPathTraversalVulnerability(t *testing.T) {
	// Create a temporary working directory
	workingDir := t.TempDir()
	ls := &LogStore{
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
	filename, err := ls.handleFileUpload(req)
	if err != nil {
		// Currently the code doesn't prevent this, but let's see what path it returns
		t.Logf("Error (expected if fixed): %v", err)
	}

	t.Logf("Returned filename: %s", filename)

	// Check if the file was written outside the working directory
	destFile := filepath.Join(ls.WorkingDir, filename)
	t.Logf("Intended destination: %s", destFile)

	// The vulnerability: the actual file might be written to:
	actualPath := filepath.Join(ls.WorkingDir, "../sensitive_file.txt")
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

// Example curl commands to test:
/*
# Vulnerable - writes outside working directory
curl -F "file=@/etc/passwd" \
  -F "file=@/dev/null" \
  --form-string "file=../../../tmp/exploit.txt" \
  http://localhost:3000/

# Or using printf to create malicious filename:
curl -F "file=@test.txt;filename=../../../tmp/escape.txt" http://localhost:3000/

# The multipart form would look like:
POST / HTTP/1.1
Content-Type: multipart/form-data; boundary=----WebKitFormBoundary

------WebKitFormBoundary
Content-Disposition: form-data; name="file"; filename="../../../etc/passwd"
Content-Type: application/octet-stream

[file contents]
------WebKitFormBoundary--
*/
