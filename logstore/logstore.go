package logstore

import (
	"context"
	"log"
	"net"
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

	// try listening on the port first and foremost
	// there's no need to do anything else if this fails

	listener := ls.getPort()

	var server http.Server
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

	log.Printf("Starting logstore on %s", listener.Addr())
	log.Printf("Storing files at: %s", ls.WorkingDir)

	if err := server.Serve(listener); err != http.ErrServerClosed {
		close(idleConnsClosed)
		return err
	}

	log.Println("Shutting down LogStore")
	<-idleConnsClosed
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

func (ls *LogStore) getPort() net.Listener {
	listener, err := net.Listen("tcp", ls.ServerAddress)
	if err != nil {
		log.Fatalf("Unrecoverable error found: %v", err)
		os.Exit(1)
	}
	return listener
}
