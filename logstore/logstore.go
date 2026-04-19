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

type LogStore struct {
	WorkingDir       string
	TempStringLength int
	MaxUploadSize    int64
}

type FileStore interface {
	Save(destination string) (string, int, error)
}

type SimpleFormStore struct {
	File     string
	Contents string
}

func (sfs *SimpleFormStore) Save(destination string) (string, int, error) {
	file_destination := filepath.Join(destination, sfs.File)
	file, err := os.Create(file_destination)
	if err != nil {
		log.Printf("File creation failed: %s", file.Name())
		log.Println(err)
		return "", 0, err
	}
	defer file.Close()

	written, err := file.WriteString(sfs.File)
	if err != nil {
		return "", 0, err
	}

	return file.Name(), written, nil
}

type MultipartFormStore struct {
	File   multipart.File
	Header multipart.FileHeader
}

func (mfs *MultipartFormStore) Save(destination string) (string, int, error) {

	safeFilename := filepath.Base(mfs.Header.Filename)
	if safeFilename == "" || safeFilename == "." || safeFilename == ".." {
		return "", 0, fmt.Errorf("invalid filename")
	}

	log.Printf("'safe' filename is: %s", safeFilename)
	log.Printf("'unsafe' filename is: %s", mfs.Header.Filename)

	file_destination := filepath.Join(destination, mfs.Header.Filename)

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

	// log.Printf("Saved file to %s", destFile)
	return file.Name(), int(written), nil
}

func (ls *LogStore) handleFileUpload(r *http.Request) (string, error) {
	if len(r.Header["Content-Type"]) < 1 {
		return "", fmt.Errorf("Content-Type is invalid; Request is invalid")
	}

	var fs FileStore

	contentType := strings.Split(r.Header["Content-Type"][0], ";")[0]
	switch contentType {
	case "application/x-www-form-urlencoded":

		if err := r.ParseForm(); err != nil {
			log.Println(err)
		}

		if !r.Form.Has("filename") || !r.Form.Has("contents") {
			return "", fmt.Errorf("contents or filename field missing in form submission")
		}

		fs = &SimpleFormStore{
			File:     r.FormValue("filename"),
			Contents: r.FormValue("contents"),
		}

	case "multipart/form-data":
		if err := r.ParseMultipartForm(ls.MaxUploadSize); err != nil {
			log.Printf("An error occured parsing the form: %v", err)
			return "", err
		}

		if r.MultipartForm == nil {
			return "", fmt.Errorf("Not a file upload")
		}

		if _, fileFieldExists := r.MultipartForm.File["file"]; !fileFieldExists {
			log.Println("'file' field doesn't exist in submission")
			return "", fmt.Errorf("File must be submitted in 'file' field and is required")
		}

		origFile, header, err := r.FormFile("file")
		if err != nil {
			return "", err
		}
		defer origFile.Close()

		fs = &MultipartFormStore{
			File:   origFile,
			Header: *header,
		}

	default:
		return "", fmt.Errorf("Unrecognized Content-Type: '%s'", contentType)
	}

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
	log.Printf("%v", ls.WorkingDir)
	cleanPath := strings.TrimPrefix(file, filepath.Join(ls.WorkingDir))
	return cleanPath, nil
}
