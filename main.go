package main

import (
	"embed"
	"fmt"
	"log"

	gosxnotifier "github.com/deckarep/gosx-notifier"
	"github.com/gen2brain/malgo"
	"github.com/getlantern/systray"
)

//go:embed assets/icon.png
var iconData embed.FS

// Store audio devices globally
var audioInputDevices []malgo.DeviceInfo

// Track device states (enabled/disabled)
var deviceStates map[string]bool

func main() {
	// Scan and log audio input devices on startup
	if err := scanAudioInputDevices(); err != nil {
		log.Printf("Error scanning audio input devices: %v", err)
	}

	// Run the app
	systray.Run(onReady, onExit)
}

func onReady() {
	// Load icon data
	iconBytes, err := iconData.ReadFile("assets/icon.png")
	if err != nil {
		log.Printf("Error loading icon: %v", err)
		// Use a default icon if custom icon fails
		iconBytes = getDefaultIcon()
	}

	// Set the icon and tooltip
	systray.SetIcon(iconBytes)
	systray.SetTitle("MicMaxer2")
	systray.SetTooltip("MicMaxer2 - Right-click for menu")

	// Create menu items
	// Note: The systray library shows menu on both left and right click
	// but we can't differentiate between them
	mShow := systray.AddMenuItem("Show Notification", "Show a notification")
	systray.AddSeparator()

	// Add audio input devices section
	if len(audioInputDevices) > 0 {
		systray.AddMenuItem("Audio Input Devices", "").Disable()
		systray.AddSeparator()

		// Initialize device states if not already done
		if deviceStates == nil {
			deviceStates = make(map[string]bool)
		}

		// Add each audio device with toggle functionality
		for _, device := range audioInputDevices {
			deviceID := device.ID.String()

			// Initialize device state (default to disabled)
			if _, exists := deviceStates[deviceID]; !exists {
				deviceStates[deviceID] = false
			}

			// Create menu item with initial state
			menuTitle := getDeviceMenuTitle(device.Name(), deviceStates[deviceID])
			deviceItem := systray.AddMenuItem(menuTitle, "Click to toggle")

			// Handle clicks in a goroutine
			go func(item *systray.MenuItem, id string, name string) {
				for range item.ClickedCh {
					// Check if going from unchecked to checked
					wasUnchecked := !deviceStates[id]

					// Toggle the state
					deviceStates[id] = !deviceStates[id]

					// Update the menu item title
					newTitle := getDeviceMenuTitle(name, deviceStates[id])
					item.SetTitle(newTitle)

					// Log the state change
					log.Printf("Device '%s' toggled to: %v", name, deviceStates[id])

					// If going from unchecked to checked, query and log the audio level
					if wasUnchecked && deviceStates[id] {
						level, err := getAudioInputLevel(id)
						if err != nil {
							log.Printf("Error getting audio level for device '%s': %v", name, err)
						} else {
							log.Printf("Audio input level for device '%s': %d", name, level)
						}
					}

					// Here you would implement the actual audio device enable/disable logic
					// For now, we're just tracking the state
				}
			}(deviceItem, deviceID, device.Name())
		}

		systray.AddSeparator()
	}

	mQuit := systray.AddMenuItem("Quit", "Quit the application")

	// Handle menu item clicks in a goroutine
	go func() {
		for {
			select {
			case <-mShow.ClickedCh:
				showNotification()
			case <-mQuit.ClickedCh:
				systray.Quit()
			}
		}
	}()
}

func onExit() {
	// Cleanup tasks go here
	log.Println("MicMaxer2 exited")
}

func showNotification() {
	// Create a new notification
	note := gosxnotifier.NewNotification("MicMaxer2 is running")

	// Set the title
	note.Title = "MicMaxer2"

	// Set the subtitle
	note.Subtitle = "Menu bar app"

	// Optionally, set a sound (you can use gosxnotifier.Default, gosxnotifier.Basso, etc.)
	note.Sound = gosxnotifier.Default

	// Push the notification
	err := note.Push()
	if err != nil {
		log.Printf("Error showing notification: %v", err)
	}
}

// getDefaultIcon returns a simple default icon as PNG bytes
func getDefaultIcon() []byte {
	// This is a minimal 16x16 PNG icon (a simple colored square)
	// In a real app, you'd want to provide a proper icon
	return []byte{
		0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00, 0x00, 0x00, 0x0d,
		0x49, 0x48, 0x44, 0x52, 0x00, 0x00, 0x00, 0x10, 0x00, 0x00, 0x00, 0x10,
		0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x91, 0x68, 0x36, 0x00, 0x00, 0x00,
		0x3a, 0x49, 0x44, 0x41, 0x54, 0x78, 0x9c, 0x62, 0xfa, 0xcf, 0xc0, 0x00,
		0x01, 0x30, 0x60, 0x00, 0x03, 0x03, 0x18, 0x58, 0xc0, 0xc0, 0xf0, 0x1f,
		0x08, 0x19, 0x10, 0x92, 0x0f, 0x62, 0x1f, 0x68, 0x61, 0x00, 0x66, 0x26,
		0x06, 0x06, 0x86, 0x3f, 0x40, 0x60, 0x60, 0x00, 0x03, 0x01, 0x86, 0x05,
		0x18, 0x19, 0x00, 0xc3, 0x00, 0x00, 0x00, 0x37, 0x2e, 0x0f, 0xf8, 0x72,
		0xc5, 0x4d, 0xe5, 0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4e, 0x44, 0xae,
		0x42, 0x60, 0x82,
	}
}

// getDeviceMenuTitle returns the menu title with appropriate state indicator
func getDeviceMenuTitle(deviceName string, enabled bool) string {
	if enabled {
		return "✓ " + deviceName
	}
	return "   " + deviceName // Three spaces to align with checkmark
}

// Alternative styling options you can use:
// Option 2: Bullet points
func getDeviceMenuTitleBullets(deviceName string, enabled bool) string {
	if enabled {
		return "● " + deviceName
	}
	return "○ " + deviceName
}

// Option 3: Status in parentheses
func getDeviceMenuTitleStatus(deviceName string, enabled bool) string {
	if enabled {
		return deviceName + " (On)"
	}
	return deviceName + " (Off)"
}

// Option 4: Square brackets
func getDeviceMenuTitleBrackets(deviceName string, enabled bool) string {
	if enabled {
		return "[✓] " + deviceName
	}
	return "[  ] " + deviceName
}

// Option 5: Emoji indicators
func getDeviceMenuTitleEmoji(deviceName string, enabled bool) string {
	if enabled {
		return "🔊 " + deviceName
	}
	return "🔇 " + deviceName
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

	// Store the devices globally
	audioInputDevices = infos

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
