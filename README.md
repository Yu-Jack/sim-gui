# sim-gui

A web-based UI for managing and exploring Harvester support bundles. This application provides an intuitive interface to upload, analyze, and navigate through support bundle contents.

## Features

- Upload and manage multiple support bundle workspaces
- Browse support bundle structure (logs, yamls, nodes, etc.)
- View resource history and changes
- Interactive file explorer
- RESTful API backend

## Installation

Build both the UI and the Go binary:

```bash
make build
```

## Usage

After building, run the binary with embedded UI:

```bash
./bin/sim-gui-linux-amd64 server
```

Options:
- `--addr`: Server address (default: `:8080`)
- `--data-dir`: Directory to store data (default: `./data`)

The server will serve both the API and the UI at `http://localhost:8080`.

## Development

For development setup and contributing guidelines, please see [CONTRIBUTING.md](CONTRIBUTING.md).