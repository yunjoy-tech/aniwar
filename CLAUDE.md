# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build System

### Windows
```bash
make.bat              # Build all services with debug symbols
start.bat             # Start all services (dapr run mode)
start.bat d           # Start with debug (starts daprd on actor server for IDE attach)
stop.bat              # Stop all daprd processes
```

### Linux/macOS
```bash
make linux             # Build for Linux
make mac              # Build for macOS
make linux_develop    # Build debug version for Linux
make start            # Start services
make stop             # Stop services
```

### Linting
```bash
make lint             # Run golangci-lint (requires ./golangci-lint binary)
```

### Other Commands
```bash
make tar              # Create tar archives
make version          # Show version info
make gmbuild         # Build GM tool (tools/gin-vue-admin)
make mod              # Update musae framework dependencies
make proto            # Pull proto definitions
```

---

## Architecture

### Microservices
This is a **Dapr-based microservices game server** built on the **Musae framework** (`../musae/`).

| Service | App Port | Out Port | PProf Port | Purpose |
|---------|-----------|-----------|-------------|---------|
| guide | 20001 | - | 20004 | New player onboarding |
| login | 21001 | 12001 | 21004 | Authentication (TapTap, Lilith, 快豹) |
| gate | 22001 | 13001 | 22004 | Client gateway, message routing |
| lobby | 23001 | - | 23004 | Matchmaking, room management |
| actor | 24001 | - | 24004 | Core game logic (User, Room, Alliance actors) |
| bill | 28001 | - | 28004 | Payment processing |
| idip | 29001 | - | 29004 | External service integration |

### Actor Pattern
The **Actor Server** implements an Actor model for game state management:
- Actor types: `UserActor`, `RoomActor`, `AllianceActor`, `CenterActor`, `MailActor`
- Each actor has isolated state and TTL-based garbage collection
- Dapr placement service runs on port 6050

### Service Startup
Services are started via `dapr.exe run` wrapper with:
- `--config`: Path to server.yaml
- `--app-id`: Service identifier
- `-p`: Application port
- `--dapr-grpc-port`: gRPC port
- `--pprof-addr`: pprof profiling port
- `--actor`: Comma-separated actor types (actor server only)

### Key Dependencies
- **Musae Framework**: `gitee.com/aniwar2/musae` (located at `../musae/`)
- **Dapr**: Distributed application runtime
- **Consul**: Service discovery (auto-started on port 8500)
- **Apollo**: Configuration center (planning migration to Nacos)
- **MongoDB**: Persistent storage
- **Redis**: Caching and session management

---

## Code Quality

### Linting (golangci-lint)
- 30+ linters enabled (see `.golangci.yml`)
- Max cyclomatic complexity: 20
- Line length limit: 160
- Blocked packages: `io/ioutil`, `github.com/gogo/protobuf`
- Requires `./golangci-lint` binary in project root

### Testing/Profiling
- Pprof available on ports 20004-29004
- Debug mode: `start.bat d` starts daprd with gRPC port 50001 for IDE attachment
- Statsviz available for real-time metrics

---

## Configuration

### Config Files
- `aniwar-server.yaml.json` - Apollo config center output
- `output/res/server.yaml` - Main server configuration
- `output/res/log.yaml` - Logging configuration
- `output/res/config-center.yaml` - Config center settings
- `output/cfg/dapr-config.yaml` - Dapr component configuration
- `output/cfg/component/` - Dapr component definitions

### Proto Definitions
- Located at `src/proto/protocol/`
- Use `go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.28.0` for protoc tools

---

## Development Workflow

### Building a Single Service
Each service is a Go package under `src/`:
```bash
cd src/<servername>
go build -o ../../output/bin/win/<servername>.exe
```

### Updating Musae Framework
```bash
make mod  # Pulls latest musae, runs go mod tidy/vendor
```

### Debugging
- Use `start.bat a` for actor server debug (daprd runs independently)
- Use `start.bat l` for lobby server debug
- Use `start.bat lo` for login server debug
- Use `start.bat g` for gate server debug
- Use `start.bat b` for bill server debug
- Use `start.bat i` for idip server debug
- Use `start.bat gu` for guide server debug

Debug mode sets gRPC port to 50001 for IDE attach.

---

## CI/CD

Jenkins integration via `script/jenkinshook.py`:
- Sends Feishu notifications on export table success/failure
- Maps SVN usernames to user IDs for @mentions
