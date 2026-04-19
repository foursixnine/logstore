package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"

	"github.com/foursixnine/logstore/logstore"
)

func main() {
	workingDir, err := os.MkdirTemp("tmp/", "logstore-workdir-*")
	ls := &logstore.LogStore{
		WorkingDir:       workingDir,
		TempStringLength: 4,
		MaxUploadSize:    32 << 20,
	}

	if err != nil {
		log.Fatal(err)
	}
	defer ls.Cleanup()
	log.Printf("Using %s as working dir", workingDir)

	var server http.Server
	server.Addr = ":3000"
	http.HandleFunc("POST /", ls.UploadFileHandler)
	http.HandleFunc("GET /", ls.IndexHandler)
	http.Handle("GET /logs/", http.StripPrefix("/logs/", http.FileServer(http.Dir(ls.WorkingDir))))

	idleConnsClosed := make(chan struct{})
	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt)
		<-sigint

		// We received an interrupt signal, shut down.
		if err := server.Shutdown(context.Background()); err != nil {
			// Error from closing listeners, or context timeout:
			log.Printf("HTTP server Shutdown: %v", err)
		}
		close(idleConnsClosed)
	}()

	log.Println("Starting logstore")
	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatal(err)
	}
	log.Println("Stopping logstore")
	<-idleConnsClosed
	log.Println("Connections stopped")
}
