package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/pkg/errors"
)

// SliceIndex finds the indx of the first element of a slice that
// holds to a predicate function
func SliceIndex(limit int, predicate func(int) bool) int {
	for i := 0; i < limit; i++ {
		if predicate(i) {
			return i
		}
	}
	return -1
}

// FormatTime returns a string representation of a time
func FormatTime(t time.Time) string {
	return fmt.Sprintf(
		"%02d:%02d:%02d %02d/%02d/%04d",
		t.Hour(), t.Minute(), t.Second(),
		t.Day(), t.Month(), t.Year(),
	)
}

// FilesAt gets a list of all of the files that are within
// a given directory specified by the RELATIVE path
// This does include all files within sub-directories
func FilesAt(path string) ([]string, error) {
	// Get the absolute path
	abs, err := filepath.Abs(path)
	if err != nil {
		return []string{}, err
	}
	result := []string{}
	err = filepath.Walk(abs, func(path string, info os.FileInfo, err error) error {
		// This ommits directories
		if info.IsDir() {
			return nil
		}
		// Get the relative path
		rel, err := filepath.Rel(abs, path)
		if err != nil {
			return err
		}
		// Add it to the results
		result = append(result, rel)
		return nil
	})
	// Handle any errors
	if err != nil {
		return []string{}, errors.Wrap(err, "couldn't walk through files")
	}
	// Return the result
	return result, nil
}
