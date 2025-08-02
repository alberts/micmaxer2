//go:build !darwin
// +build !darwin

package main

import "fmt"

// getSystemInputLevel is not implemented for non-Darwin systems
func getSystemInputLevel(deviceID string) (int, error) {
	return 0, fmt.Errorf("reading input device volume is only supported on macOS")
}
