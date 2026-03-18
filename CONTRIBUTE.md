# Contributing to LogFlux Go SDK

We love your input! We want to make contributing to LogFlux Go SDK as easy and transparent as possible, whether it's:

- Reporting a bug
- Discussing the current state of the code
- Submitting a fix
- Proposing new features
- Becoming a maintainer

## We Develop with Github

We use GitHub to host code, to track issues and feature requests, as well as accept pull requests.

## We Use [Github Flow](https://guides.github.com/introduction/flow/index.html)

Pull requests are the best way to propose changes to the codebase. We actively welcome your pull requests:

1. Fork the repo and create your branch from `main`.
2. If you've added code that should be tested, add tests.
3. If you've changed APIs, update the documentation.
4. Ensure the test suite passes.
5. Make sure your code lints.
6. Issue that pull request!

## Any contributions you make will be under the MIT Software License

In short, when you submit code changes, your submissions are understood to be under the same [MIT License](LICENSE) that covers the project. Feel free to contact the maintainers if that's a concern.

## Report bugs using Github's [issues](https://github.com/logflux-io/logflux-go-sdk/v3/issues)

We use GitHub issues to track public bugs. Report a bug by [opening a new issue](https://github.com/logflux-io/logflux-go-sdk/v3/issues/new); it's that easy!

## Write bug reports with detail, background, and sample code

**Great Bug Reports** tend to have:

- A quick summary and/or background
- Steps to reproduce
  - Be specific!
  - Give sample code if you can
- What you expected would happen
- What actually happens
- Notes (possibly including why you think this might be happening, or stuff you tried that didn't work)

## Development Process

### Prerequisites

- Go 1.22 or higher
- golangci-lint (for linting)
- Make (optional, for using Makefile commands)

### Setting Up Development Environment

1. Clone the repository:
   ```bash
   git clone https://github.com/logflux-io/logflux-go-sdk/v3.git
   cd logflux-go-sdk
   ```

2. Install dependencies:
   ```bash
   go mod download
   ```

3. Install development tools:
   ```bash
   make dev-setup
   ```

### Running Tests

Run all tests:
```bash
make test
```

Run tests with coverage:
```bash
make test-coverage
```

Run benchmarks:
```bash
make bench
```

Run end-to-end tests (requires test server):
```bash
LOGFLUX_E2E_TEST=true go test ./tests -v
```

### Code Style

We use `gofmt` and `golangci-lint` to maintain code quality:

```bash
# Format code
make fmt

# Run linter
make lint
```

### Commit Messages

We follow conventional commit messages:

- `feat:` New feature
- `fix:` Bug fix
- `docs:` Documentation changes
- `style:` Code style changes (formatting, missing semicolons, etc)
- `refactor:` Code refactoring
- `test:` Adding or updating tests
- `chore:` Maintenance tasks

Example:
```
feat: add support for custom retry policies

- Allow users to configure retry behavior
- Add exponential backoff with jitter
- Update documentation
```

## Code Review Process

1. All submissions require review before merging
2. We use GitHub pull request reviews
3. Reviewers will look for:
   - Correctness
   - Test coverage
   - Documentation
   - Code style
   - Performance implications

## Community

- Visit [LogFlux.io](https://logflux.io) for more information
- Check out our [documentation](https://docs.logflux.io)
- Join discussions in GitHub issues

## License

By contributing, you agree that your contributions will be licensed under its MIT License.

## Questions?

Feel free to open an issue with your question or reach out to the maintainers directly.