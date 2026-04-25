package logstore

import (
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/foursixnine/logstore/assets"
	"github.com/foursixnine/logstore/internal/utils"
)

var tmpl *template.Template

func initTemplates() {
	tmpl = template.Must(template.ParseFS(assets.FS, "templates/*"))
	log.Println("Initialized templates")
}

func initServer() {
	initTemplates()
	initStoreFactories()
}

type Router struct {
	http.ServeMux
	counter int
}

func NewRouter() *Router {
	return &Router{}
}

func (s *Router) HealthzHandler(w http.ResponseWriter, r *http.Request) {
	totalUploads := fmt.Sprintf("Total uploads recieved: %d", s.counter)
	io.WriteString(w, totalUploads)
}

func (s *Router) UploadFileHandler(cfg *LogStoreRuntimeConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, cfg.MaxUploadSize)

		filename, err := handleFileUpload(r, cfg)
		if err != nil {
			log.Printf("Error handling upload: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		message := fmt.Sprintf("File has been uploaded to %s%s\n", r.Host, filepath.Join("/logs", filename))
		io.WriteString(w, message)
		s.counter++
	}
}

func (s *Router) IndexHandler(cfg *LogStoreRuntimeConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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

		if err := tmpl.ExecuteTemplate(w, templateFile, data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

func handleFileUpload(r *http.Request, cfg *LogStoreRuntimeConfig) (string, error) {

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
