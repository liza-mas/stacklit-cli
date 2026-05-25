package jsonfile

import (
	"encoding/json"
	"os"
)

// MarshalIndent returns indented JSON with the POSIX-friendly trailing newline
// expected for files committed to source control.
func MarshalIndent(v any) ([]byte, error) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

// WriteIndent writes indented JSON with a trailing newline.
func WriteIndent(path string, v any, perm os.FileMode) error {
	data, err := MarshalIndent(v)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, perm)
}
