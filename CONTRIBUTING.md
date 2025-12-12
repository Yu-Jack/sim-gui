# Contributing to sim-gui

Thank you for your interest in contributing to sim-gui! This guide will help you set up your development environment and understand the development workflow.

## Prerequisites

- Go 1.22.5 or later
- Node.js 22.x
- Docker (for containerized builds)

## Building

### Quick Build

Build both the UI and the Go binary:

```bash
make build
```

This will:
1. Install UI dependencies
2. Build the UI assets
3. Copy built assets to `pkg/server/static/`
4. Build the Go binary for multiple platforms (linux-amd64, linux-arm64, darwin-arm64)
5. Output binaries to `bin/` directory

### Manual Build

Build UI only:
```bash
cd ui
npm install
npm run build
```

Build Go binary only:
```bash
go build -o bin/sim-gui main.go
```

## Development

### Backend Development

Run the server without serving static UI files (for frontend development):

```bash
go run main.go server --dev
```

Options:
- `--addr`: Server address (default: `:8080`)
- `--data-dir`: Directory to store data (default: `./data`)
- `--dev`: Enable dev mode (do not serve static files)

### Frontend Development

Start the Vite dev server for hot-reload development:

```bash
cd ui
npm run dev
```

The UI will be available at `http://localhost:5173` and will proxy API requests to `http://localhost:8080`.

Make sure the backend is running in dev mode:
```bash
go run main.go server --dev
```

### Full Stack Development Workflow

1. Terminal 1: Start backend in dev mode
   ```bash
   go run main.go server --dev
   ```

2. Terminal 2: Start frontend dev server
   ```bash
   cd ui
   npm run dev
   ```

3. Access the application at `http://localhost:5173`

## API Endpoints

### Workspace Management
- `GET /api/workspaces` - List all workspaces
- `POST /api/workspaces` - Create a new workspace
- `GET /api/workspaces/{name}` - Get workspace details
- `DELETE /api/workspaces/{name}` - Delete a workspace
- `PUT /api/workspaces/{name}` - Rename a workspace
- `POST /api/workspaces/{name}/clean-all` - Clean all workspace images
- `POST /api/workspaces/{name}/resource-history` - Get resource history
- `GET /api/workspaces/{name}/namespaces` - List namespaces
- `GET /api/workspaces/{name}/resource-types` - List resource types
- `GET /api/workspaces/{name}/resources` - Get resources

### Version Management
- `POST /api/workspaces/{name}/versions` - Upload a new version
- `POST /api/workspaces/{name}/versions/{versionID}/start` - Start simulator
- `POST /api/workspaces/{name}/versions/{versionID}/stop` - Stop simulator
- `GET /api/workspaces/{name}/versions/{versionID}/status` - Get simulator status
- `GET /api/workspaces/{name}/versions/{versionID}/kubeconfig` - Get kubeconfig
- `DELETE /api/workspaces/{name}/versions/{versionID}` - Delete a version
- `POST /api/workspaces/{name}/versions/{versionID}/clean-image` - Clean version image
- `POST /api/workspaces/{name}/versions/{versionID}/code-server` - Start code server

### Global Operations
- `POST /api/clean-all` - Clean all images

## Project Structure

```
.
├── pkg/
│   ├── server/          # HTTP server and API handlers
│   │   ├── api/         # API routes and handlers
│   │   ├── model/       # Data models
│   │   ├── store/       # Data storage layer
│   │   └── static/      # Embedded UI assets (generated)
│   ├── docker/          # Docker client utilities
│   └── kubeconfig/      # Kubeconfig utilities
├── ui/                  # React frontend application
│   ├── src/
│   │   ├── api/         # API client
│   │   ├── components/  # React components
│   │   ├── pages/       # Page components
│   │   └── types/       # TypeScript types
│   └── dist/            # Built UI assets (generated)
└── scripts/             # Build and CI scripts
```
