package logstore

import (
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type FileStore interface {
	Save(destination string) (string, int, error)
	Close() error
}

type FileStoreFactory func(r *http.Request, maxUploadSize int64) (FileStore, error)

// RegisterStoreFactory allows extending supported content types without modifying logstore.go
func RegisterStoreFactory(contentType string, factory FileStoreFactory) {
	storeFactories[contentType] = factory
}

type SimpleFormStore struct {
	File     string
	Contents string
}

func (sfs *SimpleFormStore) Close() error {
	// SimpleFormStore has no open file descriptors to close
	return nil
}

func (sfs *SimpleFormStore) Save(destination string) (string, int, error) {
	safeFilename := filepath.Base(sfs.File)
	if safeFilename == "" || strings.HasPrefix(safeFilename, ".") {
		return "", 0, fmt.Errorf("Invalid filename: %s", sfs.File)
	}

	file_destination := filepath.Join(destination, safeFilename)
	file, err := os.Create(file_destination)
	if err != nil {
		log.Printf("File creation failed: %s", file.Name())
		log.Println(err)
		return "", 0, err
	}
	defer file.Close()

	written, err := file.WriteString(sfs.Contents)
	if err != nil {
		return "", 0, err
	}

	return file.Name(), written, nil
}

func NewSimpleFormStore(r *http.Request) (*SimpleFormStore, error) {

	if err := r.ParseForm(); err != nil {
		return nil, fmt.Errorf("failed to parse form: %v", err)
	}

	if !r.Form.Has("filename") || !r.Form.Has("contents") {
		return nil, fmt.Errorf("contents or filename field missing in form submission")
	}

	fs := &SimpleFormStore{
		File:     r.FormValue("filename"),
		Contents: r.FormValue("contents"),
	}

	return fs, nil

}

type MultipartFormStore struct {
	File   multipart.File
	Header multipart.FileHeader
}

func (mfs *MultipartFormStore) Close() error {
	if mfs.File != nil {
		return mfs.File.Close()
	}
	return nil
}

func (mfs *MultipartFormStore) Save(destination string) (string, int, error) {

	safeFilename := filepath.Base(mfs.Header.Filename)
	if safeFilename == "" || safeFilename == "." || safeFilename == ".." {
		return "", 0, fmt.Errorf("invalid filename")
	}

	file_destination := filepath.Join(destination, safeFilename)

	file, err := os.Create(file_destination)
	if err != nil {
		log.Println("File creation failed")
		log.Println(err)
		return "", 0, err
	}
	defer file.Close()

	written, err := io.Copy(file, mfs.File)
	if err != nil {
		log.Println(err)
	}

	if written < mfs.Header.Size {
		log.Printf("Read less bytes (%d) than expected (%d)", written, mfs.Header.Size)
		return "", 0, fmt.Errorf("Incomplete file read")
	}

	return file.Name(), int(written), nil
}

func NewMultipartFormStore(r *http.Request, maxUploadSize int64) (*MultipartFormStore, error) {
	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		log.Printf("An error occured parsing the form: %v", err)
		return nil, err
	}

	if r.MultipartForm == nil {
		return nil, fmt.Errorf("Not a file upload")
	}

	if _, fileFieldExists := r.MultipartForm.File["file"]; !fileFieldExists {
		log.Println("'file' field doesn't exist in submission")
		return nil, fmt.Errorf("File must be submitted in 'file' field and is required")
	}

	origFile, header, err := r.FormFile("file")
	if err != nil {
		return nil, err
	}

	fs := &MultipartFormStore{
		File:   origFile,
		Header: *header,
	}

	return fs, nil
}

var storeFactories = map[string]FileStoreFactory{}

func initStoreFactories() {
	RegisterStoreFactory("application/x-www-form-urlencoded", func(r *http.Request, _ int64) (FileStore, error) {
		// We assign these explicitly to satisfy the return interface type
		fs, err := NewSimpleFormStore(r)
		return fs, err
	})

	RegisterStoreFactory("multipart/form-data", func(r *http.Request, maxUploadSize int64) (FileStore, error) {
		fs, err := NewMultipartFormStore(r, maxUploadSize)
		return fs, err
	})

	log.Println("Initialized store factories")
}
