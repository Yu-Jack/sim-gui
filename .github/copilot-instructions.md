# GitHub Copilot Instructions for sim-gui Project

## Architecture Design Principles

### Layer Separation and Responsibility

This project follows strict separation of concerns between the Docker layer and the Server/API layer:

#### Docker Layer (`pkg/docker`)
- **Purpose**: Pure Docker operations only
- **Responsibilities**:
  - Container lifecycle management (start, stop, remove)
  - Image operations (build, pull, remove)
  - Direct Docker API interactions
- **What it should NOT do**:
  - Access workspace or version data structures
  - Query or update application state/storage
  - Know about business logic or naming conventions
  - Construct instance names from workspace/version combinations

**Example**: The `Cleaner` in `pkg/docker/cleaner.go` only accepts `instanceName` as a string parameter and performs Docker cleanup operations.

#### Server/API Layer (`pkg/server/api`)
- **Purpose**: Business logic, orchestration, and state management
- **Responsibilities**:
  - Define and construct instance naming conventions (e.g., `fmt.Sprintf("%s-%s", workspaceName, versionID)`)
  - Query and update workspace/version state from storage
  - Coordinate Docker operations with application state
  - Handle HTTP endpoints and request/response logic
  - Manage version ready states

**Example**: API handlers construct `instanceName` from workspace and version before calling Docker layer methods.

### Naming Conventions

#### Instance Names
- Format: `{workspaceName}-{versionID}`
- Construction: Always done in the server/API layer
- Usage: Passed as opaque strings to Docker layer

### Code Organization Rules

1. **Docker Layer Independence**
   - No imports from `pkg/server/model` or `pkg/server/store`
   - Methods accept only primitive types or Docker-specific types
   - Example: `CleanInstance(instanceName string)` not `CleanVersion(workspaceName, versionID string)`

2. **Business Logic Location**
   - Version/workspace state management → `pkg/server/api/version_helper.go`
   - Docker orchestration → `pkg/server/api` handlers
   - Pure Docker operations → `pkg/docker`

3. **Helper Functions**
   - Workspace/Version utilities → `pkg/server/api/version_helper.go`
   - Types like `CleanVersionResult` belong in server layer, not Docker layer

### When Adding New Features

#### If adding Docker functionality:
- Place in `pkg/docker`
- Accept only instance names or Docker-specific identifiers
- No business logic or state management

#### If adding workspace/version features:
- Place orchestration logic in `pkg/server/api`
- Construct instance names in API handlers
- Use helper functions from `version_helper.go` for state management
- Call Docker layer with already-constructed instance names

### Example Patterns

#### ✅ Correct Pattern
```go
// In pkg/server/api/version.go
func (s *Server) handleCleanVersionImage(w http.ResponseWriter, r *http.Request) {
    name := r.PathValue("name")
    versionID := r.PathValue("versionID")
    instanceName := fmt.Sprintf("%s-%s", name, versionID) // Construct in server layer
    
    if err := s.cleaner.CleanInstance(instanceName); err != nil { // Pass to Docker layer
        // handle error
    }
    
    if err := s.ResetVersionReadyState(name, versionID); err != nil { // Update state in server layer
        // handle error
    }
}
```

#### ❌ Incorrect Pattern
```go
// In pkg/docker/cleaner.go - DON'T DO THIS
func (c *Cleaner) CleanVersion(workspaceName, versionID string) error {
    instanceName := fmt.Sprintf("%s-%s", workspaceName, versionID) // Business logic in Docker layer
    // ... Docker operations
    
    if err := c.resetVersionReadyState(workspaceName, versionID); err != nil { // State management in Docker layer
        return err
    }
}
```

## Summary

Keep Docker concerns in `pkg/docker` and business/orchestration concerns in `pkg/server`. The Docker layer should be agnostic of workspace/version concepts and only deal with Docker primitives like container/image names.
