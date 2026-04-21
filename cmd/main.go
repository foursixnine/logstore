package main

import (
	"flag"
	"log"
	"os"

	"github.com/foursixnine/logstore/logstore"
)

type Options struct {
	RootPath         string
	Port             string
	TempStringLength int
	MaxUploadSize    int64
}

func main() {
	var Config Options

	flag.StringVar(&Config.RootPath, "root-path", "tmp/logstore/", "Root path for the application, defaults to $PWD/tmp/logstore; path must exist")
	flag.StringVar(&Config.Port, "port", ":3000", "Port to listen on")
	flag.IntVar(&Config.TempStringLength, "string-length", 4, "Random string length, used as a name for directories to store logs")
	flag.Int64Var(&Config.MaxUploadSize, "max-upload-size", 32<<20, "Maximum upload size in bytes")
	flag.Parse()

	workingDir, err := os.MkdirTemp(Config.RootPath, "logstore-workdir-*")
	if err != nil {
		log.Fatal(err)
	}

	ls := &logstore.LogStore{
		WorkingDir:       workingDir,
		TempStringLength: Config.TempStringLength,
		MaxUploadSize:    Config.MaxUploadSize,
		ServerAddress:    Config.Port,
	}

	if err := ls.Run(); err != nil {
		log.Fatal(err)
	}

	log.Println("Server stopped successfully")
}
