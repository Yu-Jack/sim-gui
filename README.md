# sim-gui

A web-based UI for managing and exploring Harvester support bundles. This application provides an intuitive interface to upload, analyze, and navigate through support bundle contents.

## Features

### Workspace Management
- **Workspaces**: Create dedicated debugging environments for each user/customer
- **Versions**: Track multiple support bundles from the same customer over time
- **Upload Bundle**: Support single support bundle, multiple split support bundle zip files, and kubeconfig files
- **Runtime Cluster**: Connect to live clusters by uploading a kubeconfig file, enabling direct `kubectl` operations without a support bundle

### Analysis & Debugging
- **Resource Search**: Compare resources across different support bundle versions with intelligent input prompts
- **Resource History**: View and track changes between support bundle versions
- **Interactive Node Explorer**: Built-in online VS Code that automatically extracts all support bundle files - no local extraction needed

### Utilities
- **Kubeconfig Export**: Export merged kubeconfig for all active versions (both support bundles and runtime clusters) for easy integration with kubectl or k9s
- **Image Cleanup**: Clean Docker images for all workspaces or specific workspace while preserving support bundle data
- **Auto Update Notifications**: Automatically checks for new updates every hour and notifies when updates are available

### Technical Features
- RESTful API backend with embedded UI
- Docker-based simulator environment
- Persistent workspace and version storage

## Installation

Build both the UI and the Go binary:

```bash
make build
```

## Usage

After building, run the binary with embedded UI:

```bash
./scripts/start
```

Options:
- `--addr`: Server address (default: `:8080`)
- `--data-dir`: Directory to store data (default: `./data`)

The server will serve both the API and the UI at `http://localhost:8080`.

## Development

For development setup and contributing guidelines, please see [CONTRIBUTING.md](CONTRIBUTING.md).