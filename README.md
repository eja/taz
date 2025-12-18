# TAZ
**Temporary Autonomous Zone** - A lightweight, cross-platform web-based file manager for instant file sharing and management.

## Overview
TAZ is a simple, self-contained web application that allows you to quickly set up a file management interface on any supported device. Perfect for temporary file sharing, collaborative work environments, or when you need instant access to files across different devices on a network.

The Android app provides a comprehensive mobile interface, allowing you to run as a local WiFi access point, connect via BLE discovery, or scan your existing network for peers.

## Features
- **Web-based interface** - Access from any browser
- **Full file management** - Upload, download, create folders, rename, delete
- **BBS messaging system** - Optional bulletin board for team communication with audio room capability
- **Optional password protection** - Secure write operations
- **External links** - Add custom links to your file manager homepage
- **Responsive design** - Works on desktop and mobile
- **Zero dependencies** - Single binary deployment
- **Cross-platform** - Available for Linux, macOS, Windows, and Android
- **Logging support** - Optional request logging to file or stderr
- **Android app** - Full source included with BLE connectivity, WiFi hotspot, and network scanning

## Android App

### Main Menu
The app launches with a dashboard offering five main functions:
1. **WiFi Server** - Create a local WiFi access point with QR code sharing.
2. **Radio Scan** - Scan for BLE tags to automatically connect to existing TAZ instances.
3. **Network Scan** - Scan the local subnet to discover other TAZ nodes running on the network.
4. **Settings** - Configure app behavior, server name, and security.
5. **Open** - Open the embedded browser to manage your local files.

### Settings
Before choosing an operational mode, you can configure the instance:
- **Public** - Toggle between public (0.0.0.0) or private (127.0.0.1) access.
- **Bluetooth** - Enable/disable BLE credential sharing when hosting.
- **Name** - Set a custom name for your TAZ node (defaults to a random ID).
- **Password** - Set a password for create/update/delete operations.
- **Save & Return** - Applies changes and restarts the internal server.

### Operational Modes

#### WiFi Server (Hotspot Mode)
- Creates a temporary local WiFi network.
- Generates a random SSID and Password.
- Displays a status screen with two toggleable QR codes:
  - **Show WiFi QR**: Scannable credentials to join the network.
  - **Show Browser QR**: Scannable URL to open the file manager.
- **Open Local**: A shortcut to open the hosted instance in the internal view.

#### Radio Scan (BLE Client)
- Scans for Bluetooth Low Energy advertisements from other TAZ servers.
- Automatically receives WiFi credentials and IP information.
- Connects the device to the detected WiFi network and opens the interface.

#### Network Scan
- Scans the current local subnet (e.g., `192.168.1.x`) for port `35248`.
- Lists all discovered active TAZ nodes.
- One-tap connection to any discovered node.

### Permissions
The Android app requires specific permissions to function:
- **Location & Nearby Devices**: Required for BLE scanning and WiFi management.
- **Audio**: Required for the BBS audio room feature.
- **WiFi Control**: Required to create hotspots and connect to networks.

## BBS Messaging System with Audio Room
The BBS (Bulletin Board System) feature provides a simple messaging interface for team communication with added audio capability:

- **Audio Room**: Special microphone button brings participants to an audio-only room.
- **Audio Controls**: Participants can enable/disable their microphone at any time.

## Console Versions (Linux/macOS/Windows)

### Quick Start
1. Download the appropriate binary for your platform from the [Releases](../../releases/latest) page.
2. Make it executable (Unix-like systems):
   ```bash
   chmod +x taz
   ```
3. Run with default settings:
   ```bash
   ./taz
   ```
4. Open your browser to `http://localhost:35248`.

### Basic Usage Examples

#### Simple file server
```bash
./taz -root /path/to/your/files
```

#### Password-protected with custom port
```bash
./taz -password mypassword -web-port 8080
```

#### Full-featured setup
```bash
./taz \
  -password secret \
  -url "Documentation|https://example.com/docs" \
  -log \
  -log-file taz.log
```

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

Here is the complete section for command line options, including the `name` parameter used by the Android application and the external link options.

## Command Line Options

| Option | Default | Description |
|--------|---------|-------------|
| `-name` | (empty) | Set the display name for the instance/node |
| `-web-host` | `localhost` | Host address to listen on |
| `-web-port` | `35248` | Port for the web server |
| `-password` | (empty) | Password for write operations |
| `-root` | `files` | Root directory for file management |
| `-log` | `false` | Enable request logging |
| `-log-file` | (empty) | Path to log file (uses stderr if empty) |
| `-url` | (none) | External links (format: `Name\|URL`), can be used multiple times |
| `-config` | (empty) | Path to a JSON configuration file |

### External Links
You can add custom links to the homepage using the `-url` flag multiple times:
```bash
./taz \
  -url "Company Intranet|http://intranet.company.com" \
  -url "Project Repository|https://github.com/user/project" \
  -url "https://eja.tv"  # URL without custom name
```

## Building from Source

To build the console binary:
```bash
git clone https://github.com/eja/taz.git
cd taz
make
```

