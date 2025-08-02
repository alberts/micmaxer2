//go:build darwin
// +build darwin

package main

/*
#cgo LDFLAGS: -framework CoreAudio -framework Foundation
#include <CoreAudio/CoreAudio.h>
#include <stdio.h>

// Get the volume scalar for the default input device
float getDefaultInputDeviceVolume() {
    AudioDeviceID deviceID;
    UInt32 size = sizeof(AudioDeviceID);

    // Get the default input device
    OSStatus status = AudioHardwareGetProperty(
        kAudioHardwarePropertyDefaultInputDevice,
        &size,
        &deviceID
    );

    if (status != noErr || deviceID == kAudioDeviceUnknown) {
        return -1.0; // Error getting device
    }

    // Check if the device has volume control on the input scope
    Boolean hasVolumeControl = false;
    status = AudioDeviceGetPropertyInfo(
        deviceID,
        0, // master channel
        true, // input scope
        kAudioDevicePropertyVolumeScalar,
        &size,
        &hasVolumeControl
    );

    if (status != noErr || !hasVolumeControl) {
        return -1.0; // Device doesn't support volume control
    }

    // Get the volume scalar value (0.0 to 1.0)
    Float32 volume = 0.0;
    size = sizeof(Float32);
    status = AudioDeviceGetProperty(
        deviceID,
        0, // master channel
        true, // input scope
        kAudioDevicePropertyVolumeScalar,
        &size,
        &volume
    );

    if (status != noErr) {
        return -1.0; // Error getting volume
    }

    return volume;
}

// Get the mute state for the default input device
int getDefaultInputDeviceMute() {
    AudioDeviceID deviceID;
    UInt32 size = sizeof(AudioDeviceID);

    // Get the default input device
    OSStatus status = AudioHardwareGetProperty(
        kAudioHardwarePropertyDefaultInputDevice,
        &size,
        &deviceID
    );

    if (status != noErr || deviceID == kAudioDeviceUnknown) {
        return -1; // Error getting device
    }

    // Check if the device has mute control
    Boolean hasMuteControl = false;
    status = AudioDeviceGetPropertyInfo(
        deviceID,
        0, // master channel
        true, // input scope
        kAudioDevicePropertyMute,
        &size,
        &hasMuteControl
    );

    if (status != noErr || !hasMuteControl) {
        return -1; // Device doesn't support mute control
    }

    // Get the mute state
    UInt32 muted = 0;
    size = sizeof(UInt32);
    status = AudioDeviceGetProperty(
        deviceID,
        0, // master channel
        true, // input scope
        kAudioDevicePropertyMute,
        &size,
        &muted
    );

    if (status != noErr) {
        return -1; // Error getting mute state
    }

    return muted ? 1 : 0;
}
*/
import "C"
import (
	"fmt"
)

// getSystemInputLevel reads the input volume level from macOS audio device settings
// without capturing any audio. Returns a value between 0-100.
func getSystemInputLevel(deviceID string) (int, error) {
	// Get the volume scalar from Core Audio (0.0 to 1.0)
	volumeScalar := C.getDefaultInputDeviceVolume()

	if volumeScalar < 0 {
		return 0, fmt.Errorf("failed to get input device volume (device may not support volume control)")
	}

	// Check if device is muted
	muteState := C.getDefaultInputDeviceMute()
	if muteState == 1 {
		// Device is muted, return 0
		return 0, nil
	}

	// Convert from 0.0-1.0 to 0-100 scale
	volumePercent := int(volumeScalar * 100)

	// Ensure the value is within bounds
	if volumePercent < 0 {
		volumePercent = 0
	} else if volumePercent > 100 {
		volumePercent = 100
	}

	return volumePercent, nil
}
