# Contributing to PSFuzz

First off, thank you for considering contributing to PSFuzz! It's people like you that make PSFuzz such a great tool.

## Code of Conduct

By participating in this project, you are expected to uphold our Code of Conduct: be respectful and constructive.

## How Can I Contribute?

### Reporting Bugs

Before creating bug reports, please check the existing issues to avoid duplicates. When you create a bug report, include as many details as possible:

- **Use a clear and descriptive title**
- **Describe the exact steps to reproduce the problem**
- **Provide specific examples** (command line arguments, config files, etc.)
- **Describe the behavior you observed** and what you expected
- **Include screenshots** if relevant
- **Specify your environment:**
  - PSFuzz version (see banner when running `./psfuzz` or `-h`)
  - Go version (`go version`)
  - Operating system and version

**Example Bug Report:**
```
Title: Race condition when using high concurrency

Environment:
- PSFuzz: 1.0.0
- Go: 1.21.5
- OS: Ubuntu 22.04

Steps to reproduce:
1. Run: ./psfuzz -u https://example.com -c 100
2. Observe panic after ~50 requests

Expected: Should handle 100 concurrent requests
Actual: Panic with "concurrent map writes"
```

### Suggesting Enhancements

Enhancement suggestions are tracked as GitHub issues. When creating an enhancement suggestion:

- **Use a clear and descriptive title**
- **Provide a detailed description** of the suggested enhancement
- **Explain why this enhancement would be useful**
- **Include examples** of how it would work

### Adding a response module

To add a new module (e.g. a response analyzer that runs on every match): see **[MODULES.md](MODULES.md#adding-a-new-module-for-developers)**. It describes the interface, registry, output format, best practices, and points to example modules (`urlextract`, `cors`).

### Pull Requests

#### Before Submitting

1. Check existing PRs to avoid duplicates
2. Discuss significant changes in an issue first
3. Follow the coding standards (see below)
4. Ensure tests pass (if applicable)

#### Coding Standards

**Go Code Style:**
- Follow [Effective Go](https://golang.org/doc/effective_go.html)
- Run `go fmt` before committing
- Use meaningful variable and function names
  - Exported identifiers: PascalCase (e.g., `BuildTestURL`)
  - Unexported identifiers: camelCase (e.g., `buildTestURL`)
- Add comments for exported functions
- Keep functions small and focused (< 50 lines ideal)

**Commit Messages:**
- Use the present tense ("Add feature" not "Added feature")
- Use the imperative mood ("Move cursor to..." not "Moves cursor to...")
- Limit the first line to 72 characters
- Reference issues and PRs liberally

**Examples:**
```
Good:
  Add support for custom User-Agent rotation
  Fix race condition in concurrent map access
  Update README with new examples

Bad:
  Fixed stuff
  WIP
  changes
```

#### Pull Request Process

1. **Fork the repository** and create your branch from `main`
   ```bash
   git checkout -b feature/amazing-feature
   ```

2. **Make your changes**
   - Write clear, concise code
   - Add comments where necessary
   - Follow the project structure

3. **Test your changes**
   ```bash
   make build
   make test
   ./psfuzz -u https://example.com -w default -c 5
   ```

4. **Commit your changes**
   ```bash
   git commit -m "Add amazing feature"
   ```

5. **Push to your fork**
   ```bash
   git push origin feature/amazing-feature
   ```

6. **Open a Pull Request**
   - Fill in the PR template
   - Link any relevant issues
   - Provide examples/screenshots if applicable

## Development Setup

### Prerequisites

- Go 1.21 or higher (see `go.mod`)
- Git

### Setup Steps

1. **Clone your fork**
   ```bash
   git clone https://github.com/YOUR_USERNAME/PSFuzz.git
   cd PSFuzz
   ```

2. **Build the project**
   ```bash
   make build
   # or
   go build -o psfuzz .
   ```

3. **Run tests**
   ```bash
   make test
   ```

4. **Test your changes**
   ```bash
   ./psfuzz -u https://example.com -w default -c 2 -s
   ```

## Project Structure

```
PSFuzz/
├── main.go              # CLI entrypoint
├── internal/            # config, engine, filter, httpx, modules, output
├── lists/               # Wordlists (e.g. 403 bypass, useragents)
├── README.md            # Project documentation
├── CHEATSHEET.md        # Command reference
├── CONTRIBUTING.md      # This file
├── Makefile             # Build automation
├── Dockerfile           # Container image
├── RECURSION.md         # Recursion feature
└── DOCKER.md            # Docker usage
```

## Architecture Overview

The codebase is organized into packages under `internal/`:

- **config** – CLI flags, config file load/save, validation
- **engine** – scan orchestration, task queue, workers, recursion, report building
- **httpx** – HTTP client with safe-mode, redirect validation, throttling
- **filter** – response filtering (status, length, regex, duplicates)
- **output** – write report in TXT, JSON, HTML, CSV, NDJSON, compat JSON
- **modules** – response analyzers (fingerprint, cors, ai, links, etc.)

See [README.md](README.md#architecture) for the package list. When adding features, follow existing patterns (e.g. new modules in `internal/modules/`, new flags in config).

## Testing

### Manual Testing

```bash
# Basic functionality
./psfuzz -u https://example.com -w default

# Concurrency test
./psfuzz -u https://example.com -w default -c 10

# Race detection
go build -race -o psfuzz .
./psfuzz -u https://example.com -c 10

# Filter testing
./psfuzz -u https://example.com -fc 404 -s
```

### Adding Tests

PSFuzz has unit tests in `internal/*/**_test.go`. Run them with `go test ./...`. When adding features, add or extend tests in the relevant package. See [TESTING.md](TESTING.md) for the current test suites.

## Documentation

When adding features, please update:

- **README.md** – User-facing documentation
- **CHEATSHEET.md** – Command examples if CLI changes
- **Code comments** – For non-obvious logic

## Questions?

- Open an issue with the `question` label
- Reach out on Twitter: [@proviesec](https://twitter.com/proviesec)

## Recognition

Contributors will be recognized in:
- GitHub contributors list
- Release notes
- Project acknowledgments

Thank you for contributing to PSFuzz! 🚀

