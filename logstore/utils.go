package logstore

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	utils "github.com/foursixnine/logstore/internal"
)

func (ls *LogStore) Cleanup() {
	if err := os.RemoveAll(ls.WorkingDir); err != nil {
		log.Printf("Failure cleaning up %s\n", ls.WorkingDir)
		log.Fatal(err)
	}
	log.Println("Cleanup called")
}

func (ls *LogStore) createDestDir() (string, error) {
	randomString := utils.RandomString(ls.TempStringLength)
	directory := filepath.Join(ls.WorkingDir, randomString)

	for range 10 {
		if err := os.Mkdir(directory, 0755); err != nil {
			log.Printf("Error creating temporary dir (%s): %v", directory, err)
			continue
		}
		return directory, nil
	}

	return "", fmt.Errorf("Too many attempts to create temporary dir (%s)", ls.WorkingDir)
}
