# TAZ

 **Temporary Autonomous Zone** - A lightweight, cross-platform web-based file manager for instant file sharing and management.

## Overview

TAZ is a simple, self-contained web application that allows you to quickly set up a file management interface on any supported device. Perfect for temporary file sharing, collaborative work environments, or when you need instant access to files across different devices on a network.

## Features

- 🌐 **Web-based interface** - Access from any browser
- 📁 **Full file management** - Upload, download, create folders, rename, delete
- 🔒 **Optional password protection** - Secure write operations
- 🔗 **External links** - Add custom links to your file manager homepage
- 📱 **Responsive design** - Works on desktop and mobile
- ⚡ **Zero dependencies** - Single binary deployment
- 🖥️ **Cross-platform** - Available for multiple operating systems
- 📝 **Logging support** - Optional request logging to file or stderr

## Quick Start

### Download and Run

1. Download the appropriate binary for your platform from the [Releases](../../releases) page
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

#### With external links and logging
```bash
./taz \
  -password secret \
  -url "Documentation|https://example.com/docs" \
  -url "Team Chat|https://chat.example.com" \
  -log \
  -log-file access.log
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
| `-log` | `false` | Enable request logging |
| `-log-file` | (empty) | Path to log file (uses stderr if empty) |
| `-url` | (none) | External links (format: `Name\|URL`) |

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
