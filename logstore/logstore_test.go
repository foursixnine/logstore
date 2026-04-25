package logstore

import (
	"bytes"
	"log"
	"testing"
)

func TestApp(t *testing.T) {
	workingDir := t.TempDir()

	ls := &LogStore{
		WorkingDir:       workingDir,
		TempStringLength: 4,
		MaxUploadSize:    1024,
	}

	var bf bytes.Buffer
	log.SetOutput(&bf)

	go func() {
		if err := ls.Run(); err != nil {
			t.Error(err)
		}
	}()
}
