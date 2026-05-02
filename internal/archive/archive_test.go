package archive

import (
	"archive/tar"
	"io"
	"log/slog"
	"os"
	"path"
	"testing"
)

func TestMain(m *testing.M) {
	h := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug})
	slog.SetDefault(slog.New(h))
	m.Run()
}

func TestArchiveGeneration(t *testing.T) {
	workingDir := t.TempDir()

	var files = []struct {
		Name, Body string
	}{
		{"readme.txt", "This archive contains some text files."},
		{"gopher.txt", "Gopher names:\nGeorge\nGeoffrey\nGonzo"},
		{"todo.txt", "Get animal handling license."},
	}

	for file := range files {
		filePath := path.Join(workingDir, files[file].Name)
		fileBody := []byte(files[file].Body)
		tmpFile, err := os.Create(filePath)
		if err != nil {
			t.Logf("Cannot continue, %v", err)
		}
		tmpFile.Write(fileBody)
	}

	ar := NewArchive(workingDir, "test.tar")
	if err := ar.Generate(); err != nil {
		t.Fatalf("Got an error: %v", err)
	}

	tarFile, err := os.Open(ar.Name())
	if err != nil {
		t.Fatalf("Failed opening tar file: %s", err)
	}
	tr := tar.NewReader(tarFile)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break // End of archive
		}
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("Found %s (%d)\n", hdr.Name, hdr.Size)
	}

}
