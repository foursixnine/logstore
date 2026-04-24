package logstore

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/foursixnine/logstore/internal/utils"
)

func (ls *LogStore) Cleanup() {
	log.Println("Cleaned up working directory")
	if err := os.RemoveAll(ls.WorkingDir); err != nil {
		log.Printf("Failure cleaning up %s: %v\n", ls.WorkingDir, err)
	}
}

func createDestDir(cfg *LogStoreRuntimeConfig) (string, error) {
	for range 10 {
		randomString := utils.RandomString(cfg.TempStringLength)
		directory := filepath.Join(cfg.WorkingDir, randomString)

		if err := os.Mkdir(directory, 0755); err != nil {
			log.Printf("Error creating temporary dir (%s): %v", directory, err)
			continue
		}
		return directory, nil
	}

	return "", fmt.Errorf("Too many attempts to create temporary dir in working directory (%s)", cfg.WorkingDir)
}
