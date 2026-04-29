package router

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"log"
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

func init() {
	initStoreFactories()
}

type Router struct {
	http.ServeMux
	counter int
	tmpl    *template.Template
	cfg     *RouterRuntimeConfig
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

	router.HandleFunc("POST /", router.UploadFileHandler)
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
	ar.Generate()

	data, err := os.ReadFile(ar.Name())
	if err != nil {
		log.Println("Failed to read file")
		log.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	log.Printf("Serving %s", fileName)
	contentDisposition := fmt.Sprintf("attachment; filename=\"%s\"", fileName)
	w.Header().Add("Content-Disposition", contentDisposition)
	http.ServeContent(w, r, fileName, time.Now(), bytes.NewReader(data))
	log.Println("Finished serving file, deleting")
	ar.Destroy()
}

func (s *Router) UploadFileHandler(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, s.cfg.MaxUploadSize)

	filename, err := handleFileUpload(r, s.cfg)
	if err != nil {
		log.Printf("Error handling upload: %v", err)
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
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

}

func handleFileUpload(r *http.Request, cfg *RouterRuntimeConfig) (string, error) {

	if len(r.Header["Content-Type"]) < 1 {
		return "", fmt.Errorf("Content-Type is invalid; Request is invalid")
	}

	contentType := strings.Split(r.Header["Content-Type"][0], ";")[0]
	store, exists := storeFactories[contentType]
	if !exists {
		return "", fmt.Errorf("Unrecognized Content-Type: '%s'", contentType)
	}

	fs, err := store(r, cfg.MaxUploadSize)
	if err != nil {
		return "", err
	}
	defer fs.Close()

	destination, err := utils.CreateDestDir(cfg.WorkingDir, cfg.TempStringLength)
	if err != nil {
		return "", err
	}

	file, written, err := fs.Save(destination)
	if err != nil {
		os.RemoveAll(destination)
		return "", err
	}

	log.Printf("Written %d bytes to %s", written, file)
	cleanPath := strings.TrimPrefix(file, filepath.Join(cfg.WorkingDir))
	return cleanPath, nil
}
