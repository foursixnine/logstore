package logstore

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/foursixnine/logstore/internal/router"
)

type LogStore struct {
	WorkingDir       string
	TempStringLength int
	MaxUploadSize    int64
	ServerAddress    string
	CleanupDirectory bool
}

func (ls *LogStore) Run() error {
	defer ls.Cleanup()
	var server http.Server
	server.Addr = ls.ServerAddress
	server.Handler = router.NewRouter(ls.MaxUploadSize, ls.TempStringLength, ls.WorkingDir)

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

func (ls *LogStore) Cleanup() {
	if !ls.CleanupDirectory {
		log.Println("Leaving working directory intact")
		return
	}

	if err := os.RemoveAll(ls.WorkingDir); err != nil {
		log.Printf("Failure cleaning up %s: %v\n", ls.WorkingDir, err)
		return
	}

	log.Println("Cleaned up working directory")

}
