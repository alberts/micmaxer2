# MicMaxer2

A macOS menu bar application written in Go.

## Features

- Menu bar icon in the top right corner of macOS
- Left-click on the icon shows a notification
- Right-click shows a context menu with "Quit" option

## Requirements

- macOS 10.12 or later
- Go 1.19 or later
- ImageMagick (optional, for regenerating the icon)

## Building

### Quick Build

```bash
make build
```

### Manual Build

```bash
# Download dependencies
go mod download

# Build the app bundle
./build.sh
```

## Running

### From Terminal

```bash
make run
```

Or manually:
```bash
open MicMaxer2.app
```

### Install to Applications

```bash
make install
```

Or manually:
```bash
cp -r MicMaxer2.app /Applications/
```

## Development

To run the app directly without building an app bundle:

```bash
make dev
# or
go run .
```

Note: When running with `go run`, the app won't have a proper app bundle structure, so some macOS features might not work as expected.

## Project Structure

```
micmaxer2/
├── assets/
│   └── icon.png      # Menu bar icon
├── main.go           # Main application code
├── go.mod           # Go module file
├── go.sum           # Go dependencies lock file
├── build.sh         # Build script
├── Makefile         # Build automation
├── README.md        # This file
└── REQUIREMENTS.md  # Original requirements
```

## Customization

### Icon

To create a new icon, you can use ImageMagick:

```bash
convert -size 32x32 xc:transparent \
    -fill '#4A90E2' \
    -draw 'roundrectangle 4,4 28,28 4,4' \
    -fill white -font Helvetica-Bold -pointsize 18 \
    -gravity center -annotate +0+0 'M' \
    assets/icon.png
```

Or replace `assets/icon.png` with your own 32x32 PNG icon.

### Left-click Behavior

The current implementation shows a notification when the menu bar icon is left-clicked. To customize this behavior, modify the `showNotification()` function in `main.go`.

## License

This project is private and proprietary.