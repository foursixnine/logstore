package logstore

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type LogStore struct {
	WorkingDir       string
	TempStringLength int
	MaxUploadSize    int64
	ServerAddress    string
}

func (ls *LogStore) handleFileUpload(r *http.Request) (string, error) {

	if len(r.Header["Content-Type"]) < 1 {
		return "", fmt.Errorf("Content-Type is invalid; Request is invalid")
	}

	contentType := strings.Split(r.Header["Content-Type"][0], ";")[0]
	store, exists := storeFactories[contentType]
	if !exists {
		return "", fmt.Errorf("Unrecognized Content-Type: '%s'", contentType)
	}

	fs, err := store(r, ls.MaxUploadSize)
	if err != nil {
		return "", err
	}
	defer fs.Close()

	destination, err := ls.createDestDir()
	if err != nil {
		return "", err
	}

	file, written, err := fs.Save(destination)
	if err != nil {
		os.RemoveAll(destination)
		return "", err
	}

	log.Printf("Written %d bytes to %s", written, file)
	cleanPath := strings.TrimPrefix(file, filepath.Join(ls.WorkingDir))
	return cleanPath, nil
}
