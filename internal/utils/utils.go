package utils

import (
	"crypto/rand"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

var charset = []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func RandomString(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err == nil {
		for i := range b {
			b[i] = charset[int(b[i])%len(charset)]
		}
	}
	return string(b)
}

func CreateDestDir(workingDir string, tempTringLength int) (string, error) {
	for range 10 {
		randomString := RandomString(tempTringLength)
		directory := filepath.Join(workingDir, randomString)

		if err := os.Mkdir(directory, 0755); err != nil {
			log.Printf("Error creating temporary dir (%s): %v", directory, err)
			continue
		}
		return directory, nil
	}

	return "", fmt.Errorf("Too many attempts to create temporary dir in working directory (%s)", workingDir)
}
