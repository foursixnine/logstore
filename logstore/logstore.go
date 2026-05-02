package logstore

import (
	"context"
	"fmt"
	"log/slog"
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
	Server           http.Server
}

func (ls *LogStore) Run() error {
	defer ls.Cleanup()

	// try listening on the port first and foremost
	// there's no need to do anything else if this fails

	listener := ls.getPort()

	ls.Server.Handler = router.NewRouter(ls.MaxUploadSize, ls.TempStringLength, ls.WorkingDir)

	idleConnsClosed := make(chan struct{})
	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, syscall.SIGTERM, syscall.SIGINT)
		<-sigint

		if err := ls.Server.Shutdown(context.Background()); err != nil {
			slog.Error("HTTP server Shutdown", "error", err.Error())
		}
		close(idleConnsClosed)
	}()

	slog.Info("Starting logstore at:", "address", listener.Addr())
	slog.Info("Working directory:", "directory", ls.WorkingDir)

	if err := ls.Server.Serve(listener); err != http.ErrServerClosed {
		close(idleConnsClosed)
		return err
	}

	slog.Info("Shutting down LogStore")
	<-idleConnsClosed
	return nil
}

func (ls *LogStore) Cleanup() {
	if !ls.CleanupDirectory {
		slog.Info("Leaving working directory intact")
		return
	}

	if err := os.RemoveAll(ls.WorkingDir); err != nil {
		message := fmt.Sprintf("Failure cleaning up %s: %v\n", ls.WorkingDir, err)
		slog.Error(message)
		os.Exit(2)
	}

	slog.Info("Cleaned up working directory")
}

func (ls *LogStore) getPort() net.Listener {
	listener, err := net.Listen("tcp", ls.ServerAddress)
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
	return listener
}
