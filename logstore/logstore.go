package logstore

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

type LogStore struct {
	WorkingDir       string
	TempStringLength int
	MaxUploadSize    int64
	ServerAddress    string
}

type LogStoreRuntimeConfig struct {
	MaxUploadSize    int64
	TempStringLength int
	WorkingDir       string
}

func (ls *LogStore) Run() error {
	defer ls.Cleanup()
	var server http.Server
	server.Addr = ls.ServerAddress
	server.Handler = ls.SetupServer()

	idleConnsClosed := make(chan struct{})
	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, syscall.SIGTERM, syscall.SIGINT)
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

func (ls *LogStore) SetupServer() *http.ServeMux {

	cfg := &LogStoreRuntimeConfig{
		MaxUploadSize:    ls.MaxUploadSize,
		TempStringLength: ls.TempStringLength,
		WorkingDir:       ls.WorkingDir,
	}

	initServer()

	mux := http.NewServeMux()
	mux.HandleFunc("POST /", UploadFileHandler(cfg))
	mux.HandleFunc("GET /", IndexHandler(cfg))
	mux.Handle("GET /logs/", http.StripPrefix("/logs/", http.FileServer(http.Dir(ls.WorkingDir))))
	return mux

}
