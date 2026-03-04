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
- Use meaningful variable and function names (camelCase)
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
   ./psfuzz -u https://example.com -d default -c 5
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

- Go 1.16 or higher
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
   go build -o psfuzz main.go
   ```

3. **Run tests**
   ```bash
   make test
   ```

4. **Test your changes**
   ```bash
   ./psfuzz -u https://example.com -d default -c 2 -s
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

### v1.0.0 Architecture

**Core Components:**

1. **AppConfig**: Central configuration structure
   - Holds all runtime settings
   - Thread-safe access with mutexes
   - Methods for all operations

2. **Request Pipeline:**
   ```
   urlFuzzScanner() 
     → loadOrGeneratePayload()
     → scanAndFuzz()
       → sendRequest()
       → testURL()
       → responseAnalyse()
   ```

3. **Thread Safety:**
   - Use `cfg.Mutex` for shared state
   - Implement `sync.WaitGroup` for goroutines
   - Proper `defer` for resource cleanup

**Key Principles:**

- All functions are methods on `AppConfig`
- No global mutable state
- Proper error handling (no ignored errors)
- Resource cleanup with `defer`
- Type-safe flags (use `bool` not `string`)

## Testing

### Manual Testing

```bash
# Basic functionality
./psfuzz -u https://example.com -d default

# Concurrency test
./psfuzz -u https://example.com -d default -c 10

# Race detection
go build -race -o psfuzz main.go
./psfuzz -u https://example.com -c 10

# Filter testing
./psfuzz -u https://example.com -fscn 404 -s
```

### Adding Tests

While PSFuzz doesn't currently have unit tests, contributions adding them are welcome!

Example test structure:
```go
func TestAppConfig_buildTestURL(t *testing.T) {
    cfg := &AppConfig{URL: "https://example.com/"}
    result := cfg.buildTestURL("admin")
    expected := "https://example.com/admin"
    if result != expected {
        t.Errorf("Expected %s, got %s", expected, result)
    }
}
```

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

