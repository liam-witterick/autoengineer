# AutoEngineer Go Implementation

This directory contains the Go rewrite of the AutoEngineer CLI tool.

## Structure

```
go/
├── cmd/autoengineer/     # Main entry point
│   └── main.go           # CLI implementation with Cobra
├── internal/
│   ├── analysis/         # Analysis modules
│   │   ├── analysis.go   # Common analyzer interface
│   │   ├── security.go   # Security-focused analysis
│   │   ├── pipeline.go   # CI/CD pipeline analysis
│   │   └── infra.go      # Infrastructure analysis
│   ├── findings/         # Finding data structures and operations
│   │   ├── types.go      # Core Finding type
│   │   ├── filter.go     # Filtering based on ignore config
│   │   └── merge.go      # Deduplication and merging
│   ├── issues/           # GitHub issues integration
│   │   ├── create.go     # Issue creation
│   │   └── search.go     # Duplicate detection
│   ├── config/           # Configuration management
│   │   └── ignore.go     # YAML config parsing
│   └── copilot/          # Copilot CLI wrapper
│       └── client.go     # Execute copilot commands
├── go.mod                # Go module definition
└── go.sum                # Dependency checksums
```

## Building

From the repository root:

```bash
make build        # Build for current platform
make build-all    # Build for all platforms
make test         # Run tests
make install      # Install to ~/.local/bin
```

Or directly with Go:

```bash
cd go
go build -o autoengineer ./cmd/autoengineer
```

## Testing

```bash
# Run all tests
go test ./... -v

# Run tests with coverage
go test ./... -cover

# Run specific package tests
go test ./internal/findings -v
```

## Dependencies

- `github.com/spf13/cobra` - CLI framework
- `github.com/cli/go-gh/v2` - GitHub API client
- `gopkg.in/yaml.v3` - YAML parsing

## Design Principles

1. **No runtime dependencies**: Only copilot and gh CLI required
2. **Type safety**: Strong typing throughout
3. **Testability**: Interfaces and dependency injection
4. **Concurrency**: Parallel analysis when using --scope all
5. **Error handling**: Proper error wrapping and context
