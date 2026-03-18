# Project Structure Standards

## Root Directory Structure
```
/
├── Makefile              # Central build system (only one allowed)
├── go.mod               # Go module definition
├── bin/                 # Compiled binaries (gitignored)
├── tmp/                 # Temporary files and logs (gitignored)
│   └── logs/           # Test and debug logs
├── docs/               # All project documentation
│   ├── standards/      # Project standards and conventions
│   └── schemas/        # Database schema documentation
├── pkg/                # Go packages
├── examples/           # Example code
├── tests/              # Integration tests
└── scripts/            # Build and utility scripts
```

## File Organization Rules

### Binaries
- All compiled binaries go in `/bin` directory
- No binaries in subdirectories
- Build artifacts are gitignored

### Temporary Files
- All temporary files in `/tmp` directory
- Logs for testing/debugging in `/tmp/logs`
- Entire `/tmp` directory is gitignored

### Documentation
- All documentation in `/docs` directory as markdown files
- No documentation in other directories
- Standards in `/docs/standards`
- Database schemas in `/docs/schemas`

### Source Code
- Core library code in `/pkg`
- Examples in `/examples`
- Integration tests in `/tests`
- Unit tests alongside source (`*_test.go`)

## Build System
- Single `Makefile` in root directory
- No Makefiles in subdirectories
- Standard targets required:
  - `test` - Run all tests
  - `build` - Build binaries
  - `clean` - Clean build artifacts
  - `fmt` - Format code
  - `lint` - Lint code
  - `deps` - Install dependencies

## Version Control
- Use git worktrees for branch management
- Automatic merge to main after successful testing
- `.gitignore` must include `/tmp` directory
- No binaries or temporary files in version control

## AWS Integration
- Use Terraform/OpenTofu for AWS resource management
- Never manually start/stop AWS resources
- Use Amazon SSM for debugging instances
- Deployment targets: Linux ARM64