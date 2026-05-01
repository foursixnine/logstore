package utils

import (
	"errors"
	"os"
	"path"
	"strings"
	"testing"
)

func TestRandomString(t *testing.T) {
	tests := map[string]struct {
		chars              string
		expectedLengt      int
		expectedCharacters string
	}{
		"Generate 1 character": {
			chars:              "a",
			expectedLengt:      1,
			expectedCharacters: "a",
		},
		"Generate 128 character": {
			chars:              "a",
			expectedLengt:      128,
			expectedCharacters: strings.Repeat("a", 128),
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			charset = []byte(tc.chars)
			result := RandomString(tc.expectedLengt)
			if len(result) != tc.expectedLengt {
				t.Logf("String generation failed, wanted %d got %d", tc.expectedLengt, len(result))
				t.Fail()
			}
		})
	}
}

func TestCreateSessionDirectory(t *testing.T) {
	workingDir := t.TempDir()
	tests := map[string]struct {
		chars       string
		length      int
		dir         string
		maxAttempts int
	}{
		"Creates directory": {
			chars:       "a",
			length:      1,
			maxAttempts: 2,
		},
		"Too many attempts fails": {
			chars:       "a",
			length:      1,
			dir:         "a",
			maxAttempts: 2,
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			charset = []byte(tc.chars)
			maxSessionCreateAttempts = tc.maxAttempts
			if tc.dir != "" {
				os.Mkdir(path.Join(workingDir, tc.chars), 0755)
			}
			result, err := CreateSessionDirectory(workingDir, tc.length)
			if err != nil {
				if !errors.Is(err, ErrTooManyAttempts) {
					t.Logf("Got error: %v", err.Error())
					t.Fail()
				}
			}
			if !strings.HasSuffix(result, tc.chars) && err == nil {
				t.Logf("Expected %s at end of %s", tc.chars, result)
				t.Fail()
			}
		})
	}
}
