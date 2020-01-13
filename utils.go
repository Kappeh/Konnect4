package main

import (
	"fmt"
	"time"
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
