package router

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/foursixnine/logstore/assets"
	"github.com/foursixnine/logstore/internal/archive"
	"github.com/foursixnine/logstore/internal/utils"
)

type Router struct {
	http.ServeMux
	counter        int
	tmpl           *template.Template
	cfg            *RouterRuntimeConfig
	storeFactories map[string]FileStoreFactory
}

type RouterRuntimeConfig struct {
	MaxUploadSize    int64
	TempStringLength int
	WorkingDir       string
}

func NewRouter(maxUploadsize int64, tempStringLength int, workingDir string) *Router {
	router := &Router{
		tmpl: template.Must(template.ParseFS(assets.FS, "templates/*")),
	}

	router.cfg = &RouterRuntimeConfig{
		MaxUploadSize:    maxUploadsize,
		TempStringLength: tempStringLength,
		WorkingDir:       workingDir,
	}

	router.registerStoreFactory()

	router.HandleFunc("POST /", router.UploadFileHandler)
	router.HandleFunc("POST /{session}", router.UploadFileHandler)
	router.HandleFunc("GET /", router.IndexHandler)
	router.HandleFunc("GET /healthz", router.HealthzHandler)
	router.HandleFunc("GET /d/{session}/{type}/", router.ArchiveHandler)
	router.Handle("GET /logs/", http.StripPrefix("/logs/", http.FileServer(http.Dir(router.cfg.WorkingDir))))

	return router
}

func (s *Router) HealthzHandler(w http.ResponseWriter, r *http.Request) {
	totalUploads := fmt.Sprintf("Total uploads recieved: %d", s.counter)
	io.WriteString(w, totalUploads)
}

func (s *Router) ArchiveHandler(w http.ResponseWriter, r *http.Request) {
	session := r.PathValue("session")
	archiveType := r.PathValue("type")
	archivePath := path.Join(s.cfg.WorkingDir, session)
	fileName := session + "-logs." + archiveType
	ar := archive.NewArchive(archivePath, fileName)

	if err := ar.Generate(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	data, err := os.ReadFile(ar.Name())
	if err != nil {
		slog.Debug("Failed to read file", "file", ar.Name())
		slog.Error(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	slog.Info("Serving", "file", fileName)
	contentDisposition := fmt.Sprintf("attachment; filename=\"%s\"", fileName)
	w.Header().Add("Content-Disposition", contentDisposition)

	http.ServeContent(w, r, fileName, time.Now(), bytes.NewReader(data))
	slog.Info("Finished serving file, deleting")
	ar.Destroy()
}

func (s *Router) UploadFileHandler(w http.ResponseWriter, r *http.Request) {
	if len(r.Header["Content-Type"]) < 1 {
		http.Error(w, "Content-Type is invalid; Request is invalid", http.StatusBadRequest)
	}

	r.Body = http.MaxBytesReader(w, r.Body, s.cfg.MaxUploadSize)
	filename, err := s.handleFileUpload(r)
	if err != nil {
		slog.Error("Failed handling upload", "error", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	message := fmt.Sprintf("File has been uploaded to %s%s\n", r.Host, filepath.Join("/logs", filename))
	io.WriteString(w, message)
	s.counter++
}

func (s *Router) IndexHandler(w http.ResponseWriter, r *http.Request) {
	accept := r.Header.Get("Accept")
	var templateFile string

	if strings.Contains(accept, "text/html") {
		templateFile = "index.html"
	} else if accept == "application/json" {
		http.Error(w, "Json output is not yet supported", http.StatusNotImplemented)
		return
	} else {
		templateFile = "plain-text.txt"
	}

	data := map[string]template.URL{
		"Host": template.URL(r.Host),
	}

	if err := s.tmpl.ExecuteTemplate(w, templateFile, data); err != nil {
		slog.Error("Template exectution failed", "file", templateFile, "data", data)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Router) handleFileUpload(r *http.Request) (string, error) {
	contentType := strings.Split(r.Header["Content-Type"][0], ";")[0]
	store, exists := s.storeFactories[contentType]
	if !exists {
		return "", fmt.Errorf("Unrecognized Content-Type: '%s'", contentType)
	}

	fs, err := store(r, s.cfg.MaxUploadSize)
	if err != nil {
		return "", err
	}
	defer fs.Close()

	session := r.PathValue("session")
	destination, err := utils.GetSessionDirectory(s.cfg.WorkingDir, s.cfg.TempStringLength, session)
	if err != nil {
		return "", err
	}

	file, written, err := fs.Save(destination)
	if err != nil {
		os.RemoveAll(destination)
		return "", err
	}

	slog.Info("upload handled, file saved", "file", file, "bytes", written)
	cleanPath := strings.TrimPrefix(file, filepath.Join(s.cfg.WorkingDir))
	return cleanPath, nil
}

func (s *Router) GetCounter() int {
	return s.counter
}
