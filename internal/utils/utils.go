package utils

import (
	"crypto/rand"
	"errors"
	"os"
	"path"
	"path/filepath"
)

var charset = []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
var maxSessionCreateAttempts = 10
var ErrTooManyAttempts = errors.New("too many attempts to create session directory")
var ErrInvalidSession = errors.New("provided session is invalid")
var ErrSessionNotFound = errors.New("provided session not found")

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
			continue
		}
		return directory, nil
	}

	return "", ErrTooManyAttempts
}

func GetSessionDirectory(workingDir string, tempStringLength int, session string) (string, error) {
	if session != "" {
		session = filepath.Clean(session)
		sessionDirectory := path.Join(workingDir, session)
		dirInfo, err := os.Stat(sessionDirectory)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return "", ErrSessionNotFound
			}
			return "", err
		}
		if !dirInfo.IsDir() {
			return "", ErrInvalidSession
		}
		return sessionDirectory, nil
	}
	return CreateSessionDirectory(workingDir, tempStringLength)
}
