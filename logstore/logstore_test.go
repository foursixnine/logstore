package logstore

import (
	"testing"
)

func TestApp(t *testing.T) {
	workingDir := t.TempDir()

	ls := &LogStore{
		WorkingDir:       workingDir,
		TempStringLength: 4,
		MaxUploadSize:    1024,
	}

	t.Run("Server exits gracefully", func(t *testing.T) {
		go func() {
			if err := ls.Run(); err != nil {
				t.Error(err)
			}
		}()
	})
}
