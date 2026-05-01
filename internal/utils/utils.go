package utils

import (
	"crypto/rand"
	"errors"
	"log"
	"os"
	"path/filepath"
)

var charset = []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
var maxSessionCreateAttempts = 10
var ErrTooManyAttempts = errors.New("Too many attempts to create session directory")

func RandomString(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err == nil {
		for i := range b {
			b[i] = charset[int(b[i])%len(charset)]
		}
	}
	return string(b)
}

func CreateSessionDirectory(workingDir string, tempStringLength int) (string, error) {
	for range maxSessionCreateAttempts {
		randomString := RandomString(tempStringLength)
		directory := filepath.Join(workingDir, randomString)

		if err := os.Mkdir(directory, 0755); err != nil {
			log.Printf("Error creating temporary dir (%s): %v", directory, err)
			continue
		}
		return directory, nil
	}

	return "", ErrTooManyAttempts
}
