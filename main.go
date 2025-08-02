package main

import (
	"context"
	"embed"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/gen2brain/malgo"
	"github.com/getlantern/systray"
)

//go:embed assets/icon.png
var iconData embed.FS

// Constants for configuration
const (
	volumeEnforcerInterval = 60 * time.Second
	volumeResetDelay       = 1 * time.Second
	targetVolumeLevel      = 1.0 // 100%
)

// audioState manages the application's audio device state with proper synchronization
type audioState struct {
	mu                sync.RWMutex
	audioInputDevices []malgo.DeviceInfo
	deviceStates      map[string]bool
	enforcerCancel    context.CancelFunc
}

// Global audio state instance
var state = &audioState{
	deviceStates: make(map[string]bool),
}

func main() {
	// Scan and log audio input devices on startup
	if err := scanAudioInputDevices(); err != nil {
		log.Printf("Error scanning audio input devices: %v", err)
		// Continue execution even if scanning fails
	}

	// Load saved preferences and restore device states
	loadAndApplyDeviceStates()

	// Start the volume change listener
	if err := startVolumeChangeListener(); err != nil {
		log.Printf("Error starting volume change listener: %v", err)
		log.Println("Volume change events will not be monitored")
	} else {
		log.Println("Volume change listener is active - changes will be logged")
	}

	// Start the periodic volume enforcer
	startPeriodicVolumeEnforcer()

	// Run the app
	systray.Run(onReady, onExit)
}

func onReady() {
	// Load icon data
	iconBytes, err := iconData.ReadFile("assets/icon.png")
	if err != nil {
		log.Printf("Error loading icon: %v", err)
		panic(err)
	}

	// Set the icon and tooltip
	systray.SetIcon(iconBytes)
	systray.SetTitle("")
	systray.SetTooltip("MicMaxer")

	// Create menu items
	// Note: The systray library shows menu on both left and right click
	// but we can't differentiate between them

	// Add audio input devices section
	state.mu.RLock()
	devices := make([]malgo.DeviceInfo, len(state.audioInputDevices))
	copy(devices, state.audioInputDevices)
	state.mu.RUnlock()

	if len(devices) > 0 {
		systray.AddMenuItem("Audio Input Devices", "").Disable()
		systray.AddSeparator()

		// Add each audio device with toggle functionality
		for _, device := range devices {
			deviceID := device.ID.String()

			// Initialize device state (default to disabled)
			state.mu.Lock()
			if _, exists := state.deviceStates[deviceID]; !exists {
				state.deviceStates[deviceID] = false
			}
			currentState := state.deviceStates[deviceID]
			state.mu.Unlock()

			// Create menu item with initial state
			menuTitle := getDeviceMenuTitle(device.Name(), currentState)
			deviceItem := systray.AddMenuItem(menuTitle, "Click to toggle")

			// Handle clicks in a goroutine
			go func(item *systray.MenuItem, id string, name string) {
				for range item.ClickedCh {
					// Toggle the state with proper locking
					state.mu.Lock()
					wasUnchecked := !state.deviceStates[id]
					state.deviceStates[id] = !state.deviceStates[id]
					newState := state.deviceStates[id]
					state.mu.Unlock()

					// Update the menu item title
					newTitle := getDeviceMenuTitle(name, newState)
					item.SetTitle(newTitle)

					// Log the state change
					log.Printf("Device '%s' toggled to: %v", name, newState)

					// Save preferences
					saveDeviceStates()

					// If going from unchecked to checked, query and log the audio level, then set to 100%
					if wasUnchecked && newState {
						level, err := getAudioInputLevel(id)
						if err != nil {
							log.Printf("Error getting audio level for device '%s': %v", name, err)
						} else {
							log.Printf("Audio input level for device '%s': %d%%", name, level)
						}

						// Set the input level to target volume
						if err := setSystemInputLevel(targetVolumeLevel); err != nil {
							log.Printf("Error setting audio level to %d%% for device '%s': %v", int(targetVolumeLevel*100), name, err)
						} else {
							log.Printf("Successfully set audio level to %d%% for device '%s'", int(targetVolumeLevel*100), name)
						}
					}
				}
			}(deviceItem, deviceID, device.Name())
		}

		systray.AddSeparator()
	}

	mQuit := systray.AddMenuItem("Quit", "Quit the application")

	// Handle menu item clicks in a goroutine
	go func() {
		for range mQuit.ClickedCh {
			systray.Quit()
		}
	}()
}

func onExit() {
	// Stop the periodic volume enforcer
	state.mu.Lock()
	if state.enforcerCancel != nil {
		state.enforcerCancel()
	}
	state.mu.Unlock()

	// Stop the volume change listener
	if err := stopVolumeChangeListener(); err != nil {
		log.Printf("Error stopping volume change listener: %v", err)
	}

	// Cleanup tasks go here
	log.Println("MicMaxer exited")
}

// getDeviceMenuTitle returns the menu title with appropriate state indicator
func getDeviceMenuTitle(deviceName string, enabled bool) string {
	if enabled {
		return "✓ " + deviceName
	}
	return "   " + deviceName // Three spaces to align with checkmark
}

// hasSelectedDevice checks if any device is currently selected
func hasSelectedDevice() bool {
	state.mu.RLock()
	defer state.mu.RUnlock()

	for _, enabled := range state.deviceStates {
		if enabled {
			return true
		}
	}
	return false
}

// getAudioInputLevel reads the input volume level from the device settings (0-100)
func getAudioInputLevel(deviceID string) (int, error) {
	// On macOS, we use Core Audio to read the input device volume setting
	// without capturing any audio. On other platforms, this is not supported yet.
	return getSystemInputLevel(deviceID)
}

// scanAudioInputDevices scans and logs all available audio input devices
func scanAudioInputDevices() error {
	// Initialize malgo context
	ctx, err := malgo.InitContext(nil, malgo.ContextConfig{}, nil)
	if err != nil {
		return fmt.Errorf("failed to initialize audio context: %w", err)
	}
	defer func() {
		_ = ctx.Uninit()
		ctx.Free()
	}()

	// Get capture (input) devices
	infos, err := ctx.Devices(malgo.Capture)
	if err != nil {
		return fmt.Errorf("failed to get capture devices: %w", err)
	}

	// Store the devices with proper locking
	state.mu.Lock()
	state.audioInputDevices = infos
	state.mu.Unlock()

	log.Println("=== Audio Input Devices ===")
	log.Printf("Found %d audio input device(s):", len(infos))

	for i, info := range infos {
		log.Printf("  Device %d:", i+1)
		log.Printf("    Name: %s", info.Name())
		log.Printf("    ID: %s", info.ID.String())
		log.Printf("    Is Default: %v", info.IsDefault != 0)
	}

	if len(infos) == 0 {
		log.Println("  No audio input devices found")
	}

	log.Println("==========================")

	return nil
}

// loadAndApplyDeviceStates loads saved device states from preferences and applies them
func loadAndApplyDeviceStates() {

	// Load saved device IDs
	savedDeviceIDs, err := loadCheckedDevices()
	if err != nil {
		log.Printf("Error loading saved device preferences: %v", err)
		return
	}

	if len(savedDeviceIDs) == 0 {
		log.Println("No saved device preferences found")
		return
	}

	log.Printf("Loaded %d saved device preference(s)", len(savedDeviceIDs))

	// Apply saved states to existing devices
	state.mu.Lock()
	defer state.mu.Unlock()

	for _, savedID := range savedDeviceIDs {
		// Check if this device still exists
		deviceExists := false
		var deviceName string
		for _, device := range state.audioInputDevices {
			if device.ID.String() == savedID {
				deviceExists = true
				deviceName = device.Name()
				break
			}
		}

		if deviceExists {
			// Mark device as checked
			state.deviceStates[savedID] = true
			log.Printf("Restored checked state for device '%s'", deviceName)

			// Set the input level to target volume
			if err := setSystemInputLevel(targetVolumeLevel); err != nil {
				log.Printf("Error setting audio level to %d%% for device '%s': %v", int(targetVolumeLevel*100), deviceName, err)
			} else {
				log.Printf("Successfully set audio level to %d%% for device '%s'", int(targetVolumeLevel*100), deviceName)
			}
		} else {
			log.Printf("Saved device ID '%s' no longer exists on the system", savedID)
		}
	}
}

// saveDeviceStates saves the currently checked device IDs to preferences
func saveDeviceStates() {
	// Collect all checked device IDs with proper locking
	state.mu.RLock()
	var checkedDeviceIDs []string
	for id, checked := range state.deviceStates {
		if checked {
			checkedDeviceIDs = append(checkedDeviceIDs, id)
		}
	}
	state.mu.RUnlock()

	// Save to preferences
	saveCheckedDevices(checkedDeviceIDs)
	log.Printf("Saved %d checked device(s) to preferences", len(checkedDeviceIDs))
}

// startPeriodicVolumeEnforcer starts a background goroutine that periodically
// reapplies volume settings for all checked devices
func startPeriodicVolumeEnforcer() {
	ctx, cancel := context.WithCancel(context.Background())

	// Store the cancel function
	state.mu.Lock()
	state.enforcerCancel = cancel
	state.mu.Unlock()

	go func() {
		ticker := time.NewTicker(volumeEnforcerInterval)
		defer ticker.Stop()

		log.Printf("Started periodic volume enforcer - will reapply settings every %v", volumeEnforcerInterval)

		for {
			select {
			case <-ctx.Done():
				log.Println("Stopping periodic volume enforcer")
				return
			case <-ticker.C:
				enforceVolumeSettings()
			}
		}
	}()
}

// enforceVolumeSettings reapplies volume settings for all checked devices
func enforceVolumeSettings() {
	state.mu.RLock()
	// Create a copy of checked devices to avoid holding the lock during I/O operations
	checkedDevices := make(map[string]string)
	for deviceID, checked := range state.deviceStates {
		if checked {
			// Find the device name for logging
			deviceName := "Unknown"
			for _, device := range state.audioInputDevices {
				if device.ID.String() == deviceID {
					deviceName = device.Name()
					break
				}
			}
			checkedDevices[deviceID] = deviceName
		}
	}
	state.mu.RUnlock()

	// Apply volume settings without holding the lock
	for _, deviceName := range checkedDevices {
		// Set the input level to target volume
		if err := setSystemInputLevel(targetVolumeLevel); err != nil {
			log.Printf("Periodic enforcer: Error setting audio level to %d%% for device '%s': %v",
				int(targetVolumeLevel*100), deviceName, err)
		} else {
			log.Printf("Periodic enforcer: Successfully reapplied %d%% audio level for device '%s'",
				int(targetVolumeLevel*100), deviceName)
		}
	}
}
