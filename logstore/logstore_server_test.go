package logstore

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRouter(t *testing.T) {

	workingDir := t.TempDir()
	ls := &LogStore{
		WorkingDir:       workingDir,
		TempStringLength: 4,
		MaxUploadSize:    512,
	}
	server := httptest.NewServer(ls.SetupServer())
	defer server.Close()

	body, contentType := requestBodyHelper(t)

	tests := map[string]struct {
		method       string
		path         string
		code         int
		bodyContains string
		requestBody  io.Reader
		contentType  string
	}{
		"Empty GET to / returns Help text":                {"GET", server.URL + "/", 200, server.URL + "/", nil, ""},
		"Empty POST to / returns content invalid":         {"POST", server.URL + "/", 500, "Content-Type is invalid", nil, ""},
		"Valid POST to / with file returns file":          {"POST", server.URL + "/", 200, "small.txt", body, contentType},
		"Valid GET to /healthz returns number of uploads": {"GET", server.URL + "/healthz", 200, "Total uploads recieved", nil, ""},
	}

	for name, tc := range tests {

		t.Run(name, func(t *testing.T) {

			tc.method = strings.ToUpper(tc.method)
			req, err := http.NewRequest(tc.method, tc.path, tc.requestBody)
			if err != nil {
				t.Fatal(err)
			}

			if tc.contentType != "" {
				req.Header.Set("Content-Type", tc.contentType)
			}

			resp, err := server.Client().Do(req)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tc.code {
				t.Errorf("Expected status code %d, got %d", tc.code, resp.StatusCode)
			}

			if tc.bodyContains != "" {
				body, err := io.ReadAll(resp.Body)
				if err != nil {
					t.Fatal(err)
				}

				if !strings.Contains(string(body), tc.bodyContains) {
					t.Fatalf("Wanted %s in body, but was not found in \n%s", tc.bodyContains, string(body))
				}
			}
		})
	}
}

func requestBodyHelper(t *testing.T) (*bytes.Buffer, string) {
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	defer writer.Close()

	part, err := writer.CreateFormFile("file", "small.txt")
	if err != nil {
		t.Fatalf("failed to create form file: %v", err)
	}
	part.Write(bytes.Repeat([]byte("a"), 256)) // 256B

	return body, writer.FormDataContentType()
}
