package logstore

import (
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"strings"
)

func (ls *LogStore) UploadFileHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		filename, err := ls.handleFileUpload(r)
		if err != nil {
			log.Printf("Error handling upload: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		message := fmt.Sprintf("File has been uploaded to %s\n", filepath.Join("/logs", filename))
		io.WriteString(w, message)
	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}

func (ls *LogStore) IndexHandler(w http.ResponseWriter, r *http.Request) {
	accept := r.Header.Get("Accept")
	var templateFile string
	if strings.Contains(accept, "text/html") {
		templateFile = "assets/templates/index.html"
	} else if accept == "application/json" {
		http.Error(w, "Json output is not yet supported", http.StatusNotImplemented)
		return
	} else {
		templateFile = "assets/templates/plain-text.txt"
	}

	template, err := template.ParseFiles(templateFile)
	var data string
	if err != nil {
		log.Printf("%s. %v", templateFile, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	if err := template.Execute(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
