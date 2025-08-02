//go:build !darwin
// +build !darwin

package main

import "fmt"

// getSystemInputLevel is not implemented for non-Darwin systems
func getSystemInputLevel(deviceID string) (int, error) {
	return 0, fmt.Errorf("reading input device volume is only supported on macOS")
}

// startVolumeChangeListener is not implemented for non-Darwin systems
func startVolumeChangeListener() error {
	return fmt.Errorf("volume change listener is only supported on macOS")
}

// stopVolumeChangeListener is not implemented for non-Darwin systems
func stopVolumeChangeListener() error {
	return fmt.Errorf("volume change listener is only supported on macOS")
}

// setSystemInputLevel is not implemented for non-Darwin systems
func setSystemInputLevel(volume float32) error {
	return fmt.Errorf("setting input device volume is only supported on macOS")
}

// saveCheckedDevices is not implemented for non-Darwin systems
func saveCheckedDevices(deviceIDs []string) {
	// No-op on non-Darwin systems
}

// loadCheckedDevices is not implemented for non-Darwin systems
func loadCheckedDevices() ([]string, error) {
	// Return empty list on non-Darwin systems
	return []string{}, nil
}
