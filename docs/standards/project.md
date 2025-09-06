# Project Standards for LogFlux Go SDK

## Directory Structure

### Required Directories
- `/docs` - ALL documentation as markdown files
- `/docs/standards` - Project standards and conventions  
- `/docs/schemas` - Database schemas (if applicable)
- `/tmp` - Temporary files, builds, logs (must be in .gitignore)
- `/bin` - Binary outputs only, no subdirectories

### File Organization
- **Root Makefile**: Central build system, no Makefiles in subdirectories
- **Single Go module**: All Go code in one module
- **Flat structure**: Acceptable for single-purpose SDK libraries

## Build System

### Makefile Requirements
- **MUST** have central Makefile in root directory
- **NO** Makefiles in subdirectories
- **MUST** support Docker builds
- **SHOULD** include common targets:
  - `all` - Default build target
  - `build` - Build the project
  - `test` - Run tests
  - `fmt` - Format code
  - `lint` - Lint code
  - `clean` - Clean build artifacts
  - `docker-*` - Docker-based operations

### Docker Support
- **MUST** use Docker for builds when Dockerfile present
- **SHOULD** have multi-stage builds
- **MUST** use Docker for consistent CI/CD environments

### Build Artifacts
- **ALL** temporary files and builds go in `/tmp`
- **ALL** binaries go in `/bin`
- **NO** build artifacts in source directories

## Documentation Standards

### Location
- **ALL** documentation in `/docs` as markdown files
- **NO** documentation in other directories
- **Standards** in `/docs/standards/*.md`
- **Schemas** in `/docs/schemas/*.md` (if applicable)

### Content Requirements
- Project overview and purpose
- Installation instructions
- API documentation
- Configuration examples
- Best practices
- Contributing guidelines

### Format
- **MUST** use Markdown format (.md)
- **SHOULD** follow consistent structure
- **MUST** include code examples
- **SHOULD** maintain table of contents for long documents

## Version Control

### Git Requirements
- **MUST** use git for version control
- **SHOULD** use git worktrees for development
- **MUST** maintain clean .gitignore
- **SHOULD** use conventional commits

### .gitignore Requirements
- **MUST** include `/tmp/`  
- **SHOULD** include `/bin/`
- **MUST** include build artifacts
- **SHOULD** include IDE-specific files
- **MUST** include OS-generated files

## Development Workflow

### Git Worktrees
- **SHOULD** use git worktrees for parallel development
- **SHOULD** automatically merge worktrees with main when ready
- **MUST** test before merging
- **SHOULD** clean up unused worktrees

### Testing Requirements
- **MUST** test changes before committing
- **MUST** ensure builds pass before merging
- **SHOULD** run full test suite on important changes
- **MUST** maintain backwards compatibility

## Logging and Debugging

### Log Files
- **ALL** logs for testing/debugging go in `/tmp`
- **NO** logs in source directories
- **SHOULD** use structured logging
- **MUST** clean up test logs regularly

### Debug Output
- **SHOULD** support verbose/debug modes
- **MUST** not log sensitive information
- **SHOULD** include timestamp and context
- **SHOULD** be configurable via environment variables

## Configuration Management

### Configuration Files
- **SHOULD** support YAML configuration
- **SHOULD** provide sensible defaults
- **MUST** validate configuration
- **SHOULD** support environment variable overrides

### Environment Variables
- **SHOULD** use for sensitive configuration
- **MUST** document required variables
- **SHOULD** provide fallback defaults
- **SHOULD** use consistent naming convention

## Security Standards

### Secrets Management
- **NEVER** commit secrets to version control
- **SHOULD** use environment variables for secrets
- **SHOULD** support external secret management
- **MUST** document security requirements

### Authentication
- **SHOULD** support multiple authentication methods
- **MUST** use secure defaults
- **SHOULD** implement proper access controls
- **MUST** handle authentication errors gracefully

## Deployment Standards

### Target Platforms
- **Primary**: macOS (darwin-arm64) for development
- **Production**: Linux (linux-arm64) for deployment
- **SHOULD** support both platforms
- **MUST** test on target platforms

### AWS Integration
- **SHOULD** use AWS for hosting when applicable
- **SHOULD** use Terraform/OpenTofu for infrastructure
- **MUST** follow AWS security best practices
- **SHOULD** use AWS SSM for debugging instances

### Infrastructure as Code
- **SHOULD** use Terraform/OpenTofu for AWS resources
- **NO** manual AWS resource management
- **MUST** version control infrastructure code
- **SHOULD** use modules for reusable components

## Quality Standards

### Code Quality
- **MUST** pass linting and formatting checks
- **MUST** have comprehensive tests
- **SHOULD** maintain high code coverage
- **MUST** handle errors properly

### Documentation Quality
- **MUST** keep documentation up to date
- **SHOULD** include examples and tutorials
- **MUST** document breaking changes
- **SHOULD** provide migration guides when needed

### Performance
- **SHOULD** include performance benchmarks
- **MUST** avoid known performance anti-patterns
- **SHOULD** monitor resource usage
- **SHOULD** optimize for target use cases

## Maintenance

### Dependencies
- **SHOULD** keep dependencies up to date
- **MUST** review security advisories
- **SHOULD** minimize external dependencies
- **MUST** document dependency requirements

### Updates
- **SHOULD** follow semantic versioning
- **MUST** maintain backwards compatibility
- **SHOULD** provide clear upgrade paths
- **MUST** test updates thoroughly

### Monitoring
- **SHOULD** include health checks
- **SHOULD** provide metrics and observability
- **MUST** handle failures gracefully
- **SHOULD** support troubleshooting tools

## References

- [Official LogFlux Documentation](https://docs.logflux.io)
- [LogFlux Agent Configuration](https://docs.logflux.io/agent)
- [AWS Best Practices](https://aws.amazon.com/architecture/well-architected/)

## License

This project is licensed under the Apache License 2.0. See [../../LICENSE-APACHE-2.0](../../LICENSE-APACHE-2.0) for details.