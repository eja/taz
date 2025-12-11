# TAZ
**Temporary Autonomous Zone** - A lightweight, cross-platform web-based file manager for instant file sharing and management.

## Overview
TAZ is a simple, self-contained web application that allows you to quickly set up a file management interface on any supported device. Perfect for temporary file sharing, collaborative work environments, or when you need instant access to files across different devices on a network.

The Android app provides three operational modes: running as a local WiFi access point, connecting to an existing TAZ via BLE discovery, or running in standalone mode on your local network.

## Features
- **Web-based interface** - Access from any browser
- **Full file management** - Upload, download, create folders, rename, delete
- **BBS messaging system** - Optional bulletin board for team communication with audio room capability
- **Optional password protection** - Secure write operations
- **External links** - Add custom links to your file manager homepage
- **Responsive design** - Works on desktop and mobile
- **Zero dependencies** - Single binary deployment
- **Cross-platform** - Available for multiple operating systems
- **Logging support** - Optional request logging to file or stderr
- **Android app** - Three operational modes with BLE connectivity and QR code sharing

## Android App

### Main Menu
The app opens with four main options:
1. **Server** - Create a local WiFi access point with QR code sharing
2. **Client** - Scan and connect to existing TAZ instances via Bluetooth
3. **Standalone** - Run TAZ on local network without creating WiFi
4. **Settings** - Configure app behavior

### Settings
Before choosing an operational mode, configure your settings:
- **Public** Choose public (0.0.0.0) or private (127.0.0.1) access
- **Bluetooth** Enable/disable BLE credential sharing
- **Password** Set password for create/update/delete operations (read/list remain open)
- **Save & Return** Save settings and return to main menu

### Operational Modes

#### Server Mode (Local WiFi Access Point)
- Creates a temporary WiFi network with random SSID/password
- Displays QR code for easy connection
- Once server is ready, shows WiFi QR with two buttons:
  - **Show Browser QR** - Toggles to URL QR code with access URL
  - **Open Local** - Opens the embedded browser to the local TAZ
- On URL QR screen, the toggle button changes to **Show WiFi QR** to return to WiFi credentials

#### Client Mode (BLE Discovery)
- Checks for location services (required for BLE)
- Scans for BLE tags containing SSID, password, and IP information
- Automatically connects to discovered TAZ WiFi network
- Opens embedded browser to the discovered TAZ URL

#### Standalone Mode
- Traditional network operation without creating WiFi access point
- Opens embedded browser directly to `http://127.0.0.1:35248`
- Connect via existing router/network

### Permissions & Sharing
The Android app requires location permissions for BLE scanning, WiFi management permissions for hotspot functionality and Recording permission for the chat room. These permissions are requested on first launch. Additionally, the app provides a copy of its own APK file to facilitate installation on other Android devices within the network.

## BBS Messaging System with Audio Room
The BBS (Bulletin Board System) feature provides a simple messaging interface for team communication with added audio capability:

- **Enable BBS**: Use the `-bbs` flag with a database file path (e.g., `-bbs messages.db`)
- **Audio Room**: Special microphone button brings participants to an audio-only room
- **Audio Controls**: Participants can enable/disable their microphone at any time
- **Cross-platform Access**:
  - Android app: Full microphone control (speak/listen)
  - iOS/Desktop browsers: Listen-only mode available

## Console Versions (Linux/macOS/Windows)

### Quick Start
1. Download the appropriate binary for your platform from the [Releases](../../releases/latest) page
2. Make it executable (Unix-like systems):
   ```bash
   chmod +x taz
   ```
3. Run with default settings:
   ```bash
   ./taz
   ```
4. Open your browser to `http://localhost:35248`

### Basic Usage Examples

#### Simple file server
```bash
./taz -root /path/to/your/files
```

#### Password-protected with custom port
```bash
./taz -password mypassword -web-port 8080
```

#### With BBS messaging system
```bash
./taz -bbs messages.db
```

#### Full-featured setup with command-line flags
```bash
./taz \
  -password secret \
  -bbs team-messages.db \
  -url "Documentation|https://example.com/docs" \
  -url "Team Chat|https://chat.example.com" \
  -log \
  -log-file access.log
```

#### Using a configuration file
Create a `config.json` file with your settings:
```json
{
  "password": "secret",
  "bbs": "team-messages.db",
  "url": [
    "Documentation|https://example.com/docs",
    "Team Chat|https://chat.example.com"
  ],
  "log": true,
  "log_file": "access.log"
}
```
Then run TAZ, pointing it to your config file:
```bash
./taz -config config.json
```

#### Listen on all interfaces
```bash
./taz -web-host 0.0.0.0 -web-port 8080
```

## Command Line Options

| Option | Default | Description |
|--------|---------|-------------|
| `-web-host` | `localhost` | Host address to listen on |
| `-web-port` | `35248` | Port for the web server |
| `-password` | (empty) | Password for write operations |
| `-root` | `files` | Root directory for file management |
| `-bbs` | (empty) | Path to the database for BBS messaging (disabled if not provided) |
| `-log` | `false` | Enable request logging |
| `-log-file` | (empty) | Path to log file (uses stderr if empty) |
| `-url` | (none) | External links (format: `Name\|URL`), can be used multiple times |
| `-config` | (empty) | Path to a JSON configuration file. See details below. |

### Using a Configuration File (`-config`)
For complex setups, you can manage all command-line options using a JSON configuration file. This is useful for creating reusable and easily shareable configurations.

To use a configuration file, pass its path to the `-config` flag:
```bash
./taz -config my_settings.json
```

In the JSON file, the option names are the same as the command-line flags, but with hyphens (`-`) replaced by underscores (`_`).

**Example `config.json`:**
```json
{
  "web_host": "0.0.0.0",
  "web_port": 8080,
  "password": "secret",
  "root": "/shared/team-files",
  "bbs": "team-board.db",
  "log": true,
  "log_file": "taz_access.log",
  "url": [
    "Documentation|https://example.com/docs",
    "Team Chat|https://chat.example.com"
  ]
}
```

**Note:** Any options passed directly on the command line will override the values specified in the configuration file. For example:
```bash
# The web port will be 9090, overriding the value in the config file.
./taz -config config.json -web-port 9090
```

### External Links
You can add custom links to the homepage using the `-url` flag multiple times:
```bash
./taz \
  -url "Company Intranet|http://intranet.company.com" \
  -url "Project Repository|https://github.com/user/project" \
  -url "https://eja.tv"  # URL without custom name
```

## Building from Source
```bash
git clone https://github.com/eja/taz.git
cd taz
make
```
