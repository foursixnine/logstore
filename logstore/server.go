package logstore

import (
	"context"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"

	"github.com/foursixnine/logstore/assets"
)

var tmpl *template.Template

func (ls *LogStore) Run() error {
	defer ls.Cleanup()
	var server http.Server
	server.Addr = ls.ServerAddress

	tmpl = template.Must(template.ParseFS(assets.FS, "templates/*"))

	mux := http.NewServeMux()
	mux.HandleFunc("POST /", ls.UploadFileHandler)
	mux.HandleFunc("GET /", ls.IndexHandler)
	mux.Handle("GET /logs/", http.StripPrefix("/logs/", http.FileServer(http.Dir(ls.WorkingDir))))
	server.Handler = mux

	initStoreFactories()

	idleConnsClosed := make(chan struct{})
	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt)
		<-sigint

		if err := server.Shutdown(context.Background()); err != nil {
			log.Printf("HTTP server Shutdown: %v", err)
		}
		close(idleConnsClosed)
	}()

	log.Printf("Starting logstore on %s", ls.ServerAddress)
	log.Printf("Storing files at: %s", ls.WorkingDir)

	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		log.Printf("HTTP server ListenAndServe: %v", err)
		close(idleConnsClosed)
		return err
	}

	log.Println("Stopping logstore")
	<-idleConnsClosed
	log.Println("Connections stopped")
	return nil
}

func (ls *LogStore) UploadFileHandler(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, ls.MaxUploadSize)

	filename, err := ls.handleFileUpload(r)
	if err != nil {
		log.Printf("Error handling upload: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	message := fmt.Sprintf("File has been uploaded to %s%s\n", r.Host, filepath.Join("/logs", filename))
	io.WriteString(w, message)
}

func (ls *LogStore) IndexHandler(w http.ResponseWriter, r *http.Request) {
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
