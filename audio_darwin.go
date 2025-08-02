//go:build darwin
// +build darwin

package main

/*
#cgo LDFLAGS: -framework CoreAudio -framework Foundation
#include <CoreAudio/CoreAudio.h>
#include <CoreFoundation/CoreFoundation.h>
#include <stdio.h>
#include <pthread.h>

// Forward declaration of Go callback
extern void goVolumeChangeCallback(float volume, int muted);

// Mutex for thread safety
static pthread_mutex_t listenerMutex = PTHREAD_MUTEX_INITIALIZER;

// Property listener callback function
static OSStatus volumeChangeListener(
    AudioObjectID inObjectID,
    UInt32 inNumberAddresses,
    const AudioObjectPropertyAddress inAddresses[],
    void *inClientData
) {
    // Lock mutex for thread safety
    pthread_mutex_lock(&listenerMutex);

    // Get the current volume
    Float32 volume = 0.0;
    UInt32 size = sizeof(Float32);
    AudioObjectPropertyAddress volumeAddress = {
        kAudioDevicePropertyVolumeScalar,
        kAudioDevicePropertyScopeInput,
        kAudioObjectPropertyElementMain
    };

    OSStatus status = AudioObjectGetPropertyData(
        inObjectID,
        &volumeAddress,
        0,
        NULL,
        &size,
        &volume
    );

    // Get mute state
    UInt32 muted = 0;
    AudioObjectPropertyAddress muteAddress = {
        kAudioDevicePropertyMute,
        kAudioDevicePropertyScopeInput,
        kAudioObjectPropertyElementMain
    };

    size = sizeof(UInt32);
    AudioObjectGetPropertyData(
        inObjectID,
        &muteAddress,
        0,
        NULL,
        &size,
        &muted
    );

    // Call Go callback with the new values
    if (status == noErr) {
        goVolumeChangeCallback(volume, muted);
    }

    pthread_mutex_unlock(&listenerMutex);
    return noErr;
}

// Register volume change listener for the default input device
static int registerVolumeChangeListener() {
    AudioDeviceID deviceID;
    UInt32 size = sizeof(AudioDeviceID);

    // Get the default input device
    AudioObjectPropertyAddress defaultDeviceAddress = {
        kAudioHardwarePropertyDefaultInputDevice,
        kAudioObjectPropertyScopeGlobal,
        kAudioObjectPropertyElementMain
    };

    OSStatus status = AudioObjectGetPropertyData(
        kAudioObjectSystemObject,
        &defaultDeviceAddress,
        0,
        NULL,
        &size,
        &deviceID
    );

    if (status != noErr || deviceID == kAudioDeviceUnknown) {
        return -1; // Error getting device
    }

    // Register listener for volume changes
    AudioObjectPropertyAddress volumeAddress = {
        kAudioDevicePropertyVolumeScalar,
        kAudioDevicePropertyScopeInput,
        kAudioObjectPropertyElementMain
    };

    status = AudioObjectAddPropertyListener(
        deviceID,
        &volumeAddress,
        volumeChangeListener,
        NULL
    );

    if (status != noErr) {
        return -2; // Error adding volume listener
    }

    // Register listener for mute changes
    AudioObjectPropertyAddress muteAddress = {
        kAudioDevicePropertyMute,
        kAudioDevicePropertyScopeInput,
        kAudioObjectPropertyElementMain
    };

    status = AudioObjectAddPropertyListener(
        deviceID,
        &muteAddress,
        volumeChangeListener,
        NULL
    );

    if (status != noErr) {
        // Remove the volume listener if mute listener fails
        AudioObjectRemovePropertyListener(
            deviceID,
            &volumeAddress,
            volumeChangeListener,
            NULL
        );
        return -3; // Error adding mute listener
    }

    return 0; // Success
}

// Unregister volume change listener
static int unregisterVolumeChangeListener() {
    AudioDeviceID deviceID;
    UInt32 size = sizeof(AudioDeviceID);

    // Get the default input device
    AudioObjectPropertyAddress defaultDeviceAddress = {
        kAudioHardwarePropertyDefaultInputDevice,
        kAudioObjectPropertyScopeGlobal,
        kAudioObjectPropertyElementMain
    };

    OSStatus status = AudioObjectGetPropertyData(
        kAudioObjectSystemObject,
        &defaultDeviceAddress,
        0,
        NULL,
        &size,
        &deviceID
    );

    if (status != noErr || deviceID == kAudioDeviceUnknown) {
        return -1; // Error getting device
    }

    // Remove volume listener
    AudioObjectPropertyAddress volumeAddress = {
        kAudioDevicePropertyVolumeScalar,
        kAudioDevicePropertyScopeInput,
        kAudioObjectPropertyElementMain
    };

    AudioObjectRemovePropertyListener(
        deviceID,
        &volumeAddress,
        volumeChangeListener,
        NULL
    );

    // Remove mute listener
    AudioObjectPropertyAddress muteAddress = {
        kAudioDevicePropertyMute,
        kAudioDevicePropertyScopeInput,
        kAudioObjectPropertyElementMain
    };

    AudioObjectRemovePropertyListener(
        deviceID,
        &muteAddress,
        volumeChangeListener,
        NULL
    );

    return 0; // Success
}

// Get AudioDeviceID from a device UID string
static AudioDeviceID getAudioDeviceIDFromUID(const char* deviceUID) {
    if (deviceUID == NULL) {
        return kAudioDeviceUnknown;
    }

    // Create CFString from device UID
    CFStringRef uidString = CFStringCreateWithCString(NULL, deviceUID, kCFStringEncodingUTF8);
    if (uidString == NULL) {
        return kAudioDeviceUnknown;
    }

    // Get all audio devices
    AudioObjectPropertyAddress propertyAddress = {
        kAudioHardwarePropertyDevices,
        kAudioObjectPropertyScopeGlobal,
        kAudioObjectPropertyElementMain
    };

    UInt32 dataSize = 0;
    OSStatus status = AudioObjectGetPropertyDataSize(
        kAudioObjectSystemObject,
        &propertyAddress,
        0,
        NULL,
        &dataSize
    );

    if (status != noErr) {
        CFRelease(uidString);
        return kAudioDeviceUnknown;
    }

    UInt32 deviceCount = dataSize / sizeof(AudioDeviceID);
    AudioDeviceID* devices = (AudioDeviceID*)malloc(dataSize);

    status = AudioObjectGetPropertyData(
        kAudioObjectSystemObject,
        &propertyAddress,
        0,
        NULL,
        &dataSize,
        devices
    );

    if (status != noErr) {
        free(devices);
        CFRelease(uidString);
        return kAudioDeviceUnknown;
    }

    AudioDeviceID foundDevice = kAudioDeviceUnknown;

    // Check each device for matching UID
    for (UInt32 i = 0; i < deviceCount; i++) {
        // Get device UID
        propertyAddress.mSelector = kAudioDevicePropertyDeviceUID;
        propertyAddress.mScope = kAudioObjectPropertyScopeGlobal;

        CFStringRef deviceUIDStr = NULL;
        dataSize = sizeof(CFStringRef);
        status = AudioObjectGetPropertyData(
            devices[i],
            &propertyAddress,
            0,
            NULL,
            &dataSize,
            &deviceUIDStr
        );

        if (status == noErr && deviceUIDStr != NULL) {
            // Compare UIDs
            if (CFStringCompare(uidString, deviceUIDStr, 0) == kCFCompareEqualTo) {
                foundDevice = devices[i];
                CFRelease(deviceUIDStr);
                break;
            }
            CFRelease(deviceUIDStr);
        }
    }

    free(devices);
    CFRelease(uidString);
    return foundDevice;
}

// Get the volume scalar for a specific input device by UID
static float getInputDeviceVolume(const char* deviceUID) {
    AudioDeviceID deviceID = getAudioDeviceIDFromUID(deviceUID);
    if (deviceID == kAudioDeviceUnknown) {
        return -1.0; // Error getting device
    }

    // Check if the device has volume control on the input scope
    AudioObjectPropertyAddress propertyAddress = {
        kAudioDevicePropertyVolumeScalar,
        kAudioDevicePropertyScopeInput,
        kAudioObjectPropertyElementMain
    };

    Boolean hasProperty = AudioObjectHasProperty(deviceID, &propertyAddress);
    if (!hasProperty) {
        return -1.0; // Device doesn't support volume control
    }

    // Get the volume scalar value (0.0 to 1.0)
    Float32 volume = 0.0;
    UInt32 size = sizeof(Float32);
    OSStatus status = AudioObjectGetPropertyData(
        deviceID,
        &propertyAddress,
        0,
        NULL,
        &size,
        &volume
    );

    if (status != noErr) {
        return -1.0; // Error getting volume
    }

    return volume;
}

// Get the volume scalar for the default input device
static float getDefaultInputDeviceVolume() {
    AudioDeviceID deviceID;
    UInt32 size = sizeof(AudioDeviceID);

    // Get the default input device using modern API
    AudioObjectPropertyAddress propertyAddress = {
        kAudioHardwarePropertyDefaultInputDevice,
        kAudioObjectPropertyScopeGlobal,
        kAudioObjectPropertyElementMain
    };

    OSStatus status = AudioObjectGetPropertyData(
        kAudioObjectSystemObject,
        &propertyAddress,
        0,
        NULL,
        &size,
        &deviceID
    );

    if (status != noErr || deviceID == kAudioDeviceUnknown) {
        return -1.0; // Error getting device
    }

    // Check if the device has volume control on the input scope
    propertyAddress.mSelector = kAudioDevicePropertyVolumeScalar;
    propertyAddress.mScope = kAudioDevicePropertyScopeInput;
    propertyAddress.mElement = kAudioObjectPropertyElementMain;

    Boolean hasProperty = AudioObjectHasProperty(deviceID, &propertyAddress);
    if (!hasProperty) {
        return -1.0; // Device doesn't support volume control
    }

    // Get the volume scalar value (0.0 to 1.0)
    Float32 volume = 0.0;
    size = sizeof(Float32);
    status = AudioObjectGetPropertyData(
        deviceID,
        &propertyAddress,
        0,
        NULL,
        &size,
        &volume
    );

    if (status != noErr) {
        return -1.0; // Error getting volume
    }

    return volume;
}

// Get the mute state for a specific input device by UID
static int getInputDeviceMute(const char* deviceUID) {
    AudioDeviceID deviceID = getAudioDeviceIDFromUID(deviceUID);
    if (deviceID == kAudioDeviceUnknown) {
        return -1; // Error getting device
    }

    // Check if the device has mute control
    AudioObjectPropertyAddress propertyAddress = {
        kAudioDevicePropertyMute,
        kAudioDevicePropertyScopeInput,
        kAudioObjectPropertyElementMain
    };

    Boolean hasProperty = AudioObjectHasProperty(deviceID, &propertyAddress);
    if (!hasProperty) {
        return -1; // Device doesn't support mute control
    }

    // Get the mute state
    UInt32 muted = 0;
    UInt32 size = sizeof(UInt32);
    OSStatus status = AudioObjectGetPropertyData(
        deviceID,
        &propertyAddress,
        0,
        NULL,
        &size,
        &muted
    );

    if (status != noErr) {
        return -1; // Error getting mute state
    }

    return muted ? 1 : 0;
}

// Get the mute state for the default input device
static int getDefaultInputDeviceMute() {
    AudioDeviceID deviceID;
    UInt32 size = sizeof(AudioDeviceID);

    // Get the default input device using modern API
    AudioObjectPropertyAddress propertyAddress = {
        kAudioHardwarePropertyDefaultInputDevice,
        kAudioObjectPropertyScopeGlobal,
        kAudioObjectPropertyElementMain
    };

    OSStatus status = AudioObjectGetPropertyData(
        kAudioObjectSystemObject,
        &propertyAddress,
        0,
        NULL,
        &size,
        &deviceID
    );

    if (status != noErr || deviceID == kAudioDeviceUnknown) {
        return -1; // Error getting device
    }

    // Check if the device has mute control
    propertyAddress.mSelector = kAudioDevicePropertyMute;
    propertyAddress.mScope = kAudioDevicePropertyScopeInput;
    propertyAddress.mElement = kAudioObjectPropertyElementMain;

    Boolean hasProperty = AudioObjectHasProperty(deviceID, &propertyAddress);
    if (!hasProperty) {
        return -1; // Device doesn't support mute control
    }

    // Get the mute state
    UInt32 muted = 0;
    size = sizeof(UInt32);
    status = AudioObjectGetPropertyData(
        deviceID,
        &propertyAddress,
        0,
        NULL,
        &size,
        &muted
    );

    if (status != noErr) {
        return -1; // Error getting mute state
    }

    return muted ? 1 : 0;
}

// Set the volume scalar for a specific input device by UID
static int setInputDeviceVolume(const char* deviceUID, float volume) {
    AudioDeviceID deviceID = getAudioDeviceIDFromUID(deviceUID);
    if (deviceID == kAudioDeviceUnknown) {
        return -1; // Error getting device
    }

    // Set up the property address for volume
    AudioObjectPropertyAddress propertyAddress = {
        kAudioDevicePropertyVolumeScalar,
        kAudioDevicePropertyScopeInput,
        kAudioObjectPropertyElementMain
    };

    // Check if the device has volume control
    Boolean hasProperty = AudioObjectHasProperty(deviceID, &propertyAddress);
    if (!hasProperty) {
        return -2; // Device doesn't support volume control
    }

    // Set the volume scalar value (0.0 to 1.0)
    Float32 volumeValue = volume;
    OSStatus status = AudioObjectSetPropertyData(
        deviceID,
        &propertyAddress,
        0,
        NULL,
        sizeof(Float32),
        &volumeValue
    );

    if (status != noErr) {
        return -3; // Error setting volume
    }

    return 0; // Success
}

// Set the volume scalar for the default input device
static int setDefaultInputDeviceVolume(float volume) {
    AudioDeviceID deviceID;
    UInt32 size = sizeof(AudioDeviceID);

    // Get the default input device
    AudioObjectPropertyAddress propertyAddress = {
        kAudioHardwarePropertyDefaultInputDevice,
        kAudioObjectPropertyScopeGlobal,
        kAudioObjectPropertyElementMain
    };

    OSStatus status = AudioObjectGetPropertyData(
        kAudioObjectSystemObject,
        &propertyAddress,
        0,
        NULL,
        &size,
        &deviceID
    );

    if (status != noErr || deviceID == kAudioDeviceUnknown) {
        return -1; // Error getting device
    }

    // Set up the property address for volume
    propertyAddress.mSelector = kAudioDevicePropertyVolumeScalar;
    propertyAddress.mScope = kAudioDevicePropertyScopeInput;
    propertyAddress.mElement = kAudioObjectPropertyElementMain;

    // Check if the device has volume control
    Boolean hasProperty = AudioObjectHasProperty(deviceID, &propertyAddress);
    if (!hasProperty) {
        return -2; // Device doesn't support volume control
    }

    // Set the volume scalar value (0.0 to 1.0)
    Float32 volumeValue = volume;
    status = AudioObjectSetPropertyData(
        deviceID,
        &propertyAddress,
        0,
        NULL,
        sizeof(Float32),
        &volumeValue
    );

    if (status != noErr) {
        return -3; // Error setting volume
    }

    return 0; // Success
}

// Save checked device IDs to user preferences
static void saveCheckedDevices(const char** deviceIDs, int count) {
    // Create the app ID for preferences
    CFStringRef appID = CFStringCreateWithCString(NULL, "com.micmaxer2.app", kCFStringEncodingUTF8);
    CFStringRef key = CFStringCreateWithCString(NULL, "CheckedAudioDevices", kCFStringEncodingUTF8);

    if (count == 0) {
        // Remove the preference if no devices are checked
        CFPreferencesSetAppValue(key, NULL, appID);
    } else {
        // Create an array to store device IDs
        CFMutableArrayRef deviceArray = CFArrayCreateMutable(NULL, count, &kCFTypeArrayCallBacks);

        for (int i = 0; i < count; i++) {
            CFStringRef deviceID = CFStringCreateWithCString(NULL, deviceIDs[i], kCFStringEncodingUTF8);
            CFArrayAppendValue(deviceArray, deviceID);
            CFRelease(deviceID);
        }

        // Save to preferences
        CFPreferencesSetAppValue(key, deviceArray, appID);
        CFRelease(deviceArray);
    }

    // Synchronize to disk
    CFPreferencesAppSynchronize(appID);

    CFRelease(key);
    CFRelease(appID);
}

// Load checked device IDs from user preferences
// Returns the number of device IDs loaded, -1 on error
// Caller must free the returned strings and array
static int loadCheckedDevices(char*** deviceIDs) {
    CFStringRef appID = CFStringCreateWithCString(NULL, "com.micmaxer2.app", kCFStringEncodingUTF8);
    CFStringRef key = CFStringCreateWithCString(NULL, "CheckedAudioDevices", kCFStringEncodingUTF8);

    // Get the array from preferences
    CFArrayRef deviceArray = (CFArrayRef)CFPreferencesCopyAppValue(key, appID);

    CFRelease(key);
    CFRelease(appID);

    if (deviceArray == NULL) {
        return 0; // No saved preferences
    }

    // Check if it's actually an array
    if (CFGetTypeID(deviceArray) != CFArrayGetTypeID()) {
        CFRelease(deviceArray);
        return -1; // Invalid data type
    }

    CFIndex count = CFArrayGetCount(deviceArray);
    if (count == 0) {
        CFRelease(deviceArray);
        return 0;
    }

    // Allocate memory for device IDs
    *deviceIDs = (char**)malloc(count * sizeof(char*));

    int validCount = 0;
    for (CFIndex i = 0; i < count; i++) {
        CFStringRef deviceID = (CFStringRef)CFArrayGetValueAtIndex(deviceArray, i);
        if (CFGetTypeID(deviceID) == CFStringGetTypeID()) {
            // Convert CFString to C string
            CFIndex length = CFStringGetLength(deviceID);
            CFIndex maxSize = CFStringGetMaximumSizeForEncoding(length, kCFStringEncodingUTF8) + 1;
            char* buffer = (char*)malloc(maxSize);

            if (CFStringGetCString(deviceID, buffer, maxSize, kCFStringEncodingUTF8)) {
                (*deviceIDs)[validCount] = buffer;
                validCount++;
            } else {
                free(buffer);
            }
        }
    }

    CFRelease(deviceArray);
    return validCount;
}
*/
import "C"
import (
	"fmt"
	"log"
	"unsafe"
)

// goVolumeChangeCallback is called from C when volume or mute state changes
//
//export goVolumeChangeCallback
func goVolumeChangeCallback(volume C.float, muted C.int) {
	// Convert C types to Go types
	volumeFloat := float32(volume)
	isMuted := muted != 0

	// Convert volume from 0.0-1.0 to 0-100 scale
	volumePercent := int(volumeFloat * 100)

	// Log the change
	if isMuted {
		log.Printf("[Volume Change Event] Input device is MUTED (volume setting: %d%%)", volumePercent)
	} else {
		log.Printf("[Volume Change Event] Input level changed to: %d%%", volumePercent)
	}

	// Check if any device is selected in the menu
	if hasSelectedDevice() && volumeFloat < 1.0 && !isMuted {
		log.Printf("[Volume Change Event] Detected change on monitored device - resetting to 100%%")

		// Schedule volume reset in a non-blocking goroutine
		go func() {
			// Set the volume back to 100% on the default device
			if err := setSystemInputLevel("", 1.0); err != nil {
				log.Printf("[Volume Change Event] Error resetting volume to 100%%: %v", err)
			} else {
				log.Printf("[Volume Change Event] Successfully reset volume to 100%%")
			}
		}()
	}
}

// startVolumeChangeListener registers a listener for volume change events
func startVolumeChangeListener() error {
	result := C.registerVolumeChangeListener()
	switch result {
	case 0:
		log.Println("Successfully registered volume change listener")
		return nil
	case -1:
		return fmt.Errorf("failed to get default input device")
	case -2:
		return fmt.Errorf("failed to register volume change listener")
	case -3:
		return fmt.Errorf("failed to register mute change listener")
	default:
		return fmt.Errorf("unknown error registering listener: %d", result)
	}
}

// stopVolumeChangeListener unregisters the volume change listener
func stopVolumeChangeListener() error {
	result := C.unregisterVolumeChangeListener()
	if result != 0 {
		return fmt.Errorf("failed to unregister volume change listener")
	}
	log.Println("Successfully unregistered volume change listener")
	return nil
}

// getSystemInputLevel reads the input volume level from macOS audio device settings
// without capturing any audio. Returns a value between 0-100.
func getSystemInputLevel(deviceID string) (int, error) {
	var volumeScalar C.float
	var muteState C.int

	// If deviceID is empty, use the default device
	if deviceID == "" {
		volumeScalar = C.getDefaultInputDeviceVolume()
		muteState = C.getDefaultInputDeviceMute()
	} else {
		// Convert deviceID to C string and get volume for specific device
		cDeviceID := C.CString(deviceID)
		defer C.free(unsafe.Pointer(cDeviceID))

		volumeScalar = C.getInputDeviceVolume(cDeviceID)
		muteState = C.getInputDeviceMute(cDeviceID)
	}

	if volumeScalar < 0 {
		return 0, fmt.Errorf("failed to get input device volume (device may not support volume control)")
	}

	// Check if device is muted
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

// setSystemInputLevel sets the input volume level for a specific input device
// The deviceID parameter specifies which device to control (empty string for default device)
// The volume parameter should be between 0.0 and 1.0
func setSystemInputLevel(deviceID string, volume float32) error {
	// Ensure volume is within bounds
	if volume < 0.0 {
		volume = 0.0
	} else if volume > 1.0 {
		volume = 1.0
	}

	var result C.int

	// If deviceID is empty, use the default device
	if deviceID == "" {
		result = C.setDefaultInputDeviceVolume(C.float(volume))
	} else {
		// Convert deviceID to C string and set volume for specific device
		cDeviceID := C.CString(deviceID)
		defer C.free(unsafe.Pointer(cDeviceID))

		result = C.setInputDeviceVolume(cDeviceID, C.float(volume))
	}

	switch result {
	case 0:
		return nil // Success
	case -1:
		return fmt.Errorf("failed to get device")
	case -2:
		return fmt.Errorf("device doesn't support volume control")
	case -3:
		return fmt.Errorf("failed to set volume")
	default:
		return fmt.Errorf("unknown error setting volume: %d", result)
	}
}

// saveCheckedDevices saves the list of checked device IDs to user preferences
func saveCheckedDevices(deviceIDs []string) {
	if len(deviceIDs) == 0 {
		// Call with NULL to remove the preference
		C.saveCheckedDevices(nil, 0)
		return
	}

	// Convert Go strings to C strings
	cDeviceIDs := make([]*C.char, len(deviceIDs))
	for i, id := range deviceIDs {
		cDeviceIDs[i] = C.CString(id)
	}

	// Call the C function
	C.saveCheckedDevices(&cDeviceIDs[0], C.int(len(deviceIDs)))

	// Free the C strings
	for _, cStr := range cDeviceIDs {
		C.free(unsafe.Pointer(cStr))
	}
}

// loadCheckedDevices loads the list of checked device IDs from user preferences
func loadCheckedDevices() ([]string, error) {
	var cDeviceIDs **C.char
	count := C.loadCheckedDevices(&cDeviceIDs)

	if count < 0 {
		return nil, fmt.Errorf("failed to load preferences")
	}

	if count == 0 {
		return []string{}, nil
	}

	// Convert C strings to Go strings
	deviceIDs := make([]string, count)

	// Create a slice from the C array
	cSlice := (*[1 << 30]*C.char)(unsafe.Pointer(cDeviceIDs))[:count:count]

	for i := 0; i < int(count); i++ {
		deviceIDs[i] = C.GoString(cSlice[i])
		// Free the C string
		C.free(unsafe.Pointer(cSlice[i]))
	}

	// Free the array itself
	C.free(unsafe.Pointer(cDeviceIDs))

	return deviceIDs, nil
}
