# OpenCrab Electron Desktop App

This directory contains the Electron wrapper for OpenCrab, providing a native desktop application with system tray support for Windows, macOS, and Linux.

## Prerequisites

### 1. Go Binary (Required)
The Electron app requires the compiled Go binary to function. You have two options:

**Option A: Use existing binary (without Go installed)**
```bash
# If you have a pre-built binary
cp ../opencrab-macos ../opencrab
```

**Option B: Build from source (requires Go)**
Build the backend binary in the project root first.

### 2. Electron Dependencies
```bash
cd electron
npm install
```

## Development

Run the app in development mode:
```bash
npm run dev-app
```

This expects:
- Go backend on port 3000
- Frontend dev server on port 5173

## Building for Production

### Quick Build
```bash
# Ensure Go binary exists in parent directory
ls ../opencrab  # Should exist

# Build for current platform
npm run build

# Platform-specific builds
npm run build:mac
npm run build:win
npm run build:linux
```

### Build Output
- Built applications are in `electron/dist/`
- macOS: `.dmg` and `.zip`
- Windows: installer and portable exe
- Linux: `.AppImage` and `.deb`

## Configuration

### Port
Default port is 3000. To change, edit `main.js`:
```javascript
const PORT = 3000;
```

### Database Location
- **Development**: `../data/opencrab.db`
- **Production**:
  - macOS: `~/Library/Application Support/OpenCrab/data/`
  - Windows: `%APPDATA%/OpenCrab/data/`
  - Linux: `~/.config/OpenCrab/data/`
