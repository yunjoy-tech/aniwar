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
# Download golangci-lint binary to project root (version must match .golangci.yml's go version)
curl -sSfL https://raw.githubusercontent.com/golangci-lint/golangci-lint/master/install.sh | sh -s -- -b . v1.57.2

make lint             # Run golangci-lint
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
- **Musae Framework**: `github.com/yunjoy-tech/musae` (located at `../musae/`)
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
```bash
go test ./src/... -run TestFuncName    # Run single test
go test ./src/... -v                   # Verbose output
go test ./src/... -count=1             # Disable cache
```
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
```bash
# Install protoc tools
go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.28.0
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

make proto            # Pull latest proto definitions
```

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
Debug mode starts daprd separately with gRPC port 50001 for IDE attachment.

| Command | Service | IDE Launch Config Example |
|---------|---------|--------------------------|
| `start.bat a` | actor | `appid=actor actor=user inaddr=24001 gport=50001` |
| `start.bat l` | lobby | `appid=lobby inaddr=23001 gport=50001` |
| `start.bat lo` | login | `appid=login inaddr=21001 gport=50001` |
| `start.bat g` | gate | `appid=gate9999 outaddr=13001 inaddr=22001 gport=50001` |
| `start.bat b` | bill | `appid=bill inaddr=28001 gport=50001` |
| `start.bat i` | idip | `appid=idip inaddr=29001 gport=50001` |
| `start.bat gu` | guide | `appid=guide inaddr=20001 gport=50001` |

### Infrastructure Ports
- Consul UI: `http://localhost:8500`
- Dapr Placement: `port 6050`

---

## CI/CD

Jenkins integration via `script/jenkinshook.py`:
- Sends Feishu notifications on export table success/failure
- Maps SVN usernames to user IDs for @mentions
