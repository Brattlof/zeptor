# Contributing to Zeptor

Thanks for your interest in contributing to Zeptor! This document provides guidelines and instructions for contributing.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Setup](#development-setup)
- [Making Changes](#making-changes)
- [Testing](#testing)
- [Pull Request Process](#pull-request-process)
- [Coding Standards](#coding-standards)

## Code of Conduct

This project adheres to the [Contributor Covenant Code of Conduct](CODE_OF_CONDUCT.md). By participating, you are expected to uphold this code.

## Getting Started

### Prerequisites

- Go 1.23 or later
- Docker (for eBPF development)
- Make
- clang/llvm (for eBPF compilation)
- Linux kernel 5.4+ (for eBPF features)

### Development Setup

1. Fork the repository
2. Clone your fork:
   ```bash
   git clone https://github.com/<your-username>/zeptor.git
   cd zeptor
   ```
3. Install dependencies:
   ```bash
   make install-deps
   ```
4. Build the project:
   ```bash
   make build
   ```
5. Run tests:
   ```bash
   make test
   ```

## Making Changes

### Branch Naming

Use descriptive branch names with prefixes:
- `feat/` - New features
- `fix/` - Bug fixes
- `docs/` - Documentation changes
- `refactor/` - Code refactoring
- `test/` - Test additions/modifications
- `chore/` - Maintenance tasks

### Commit Messages

Follow [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>(<scope>): <description>

[optional body]

[optional footer(s)]
```

Types: `feat`, `fix`, `docs`, `style`, `refactor`, `test`, `chore`, `perf`, `ci`, `build`

**Examples:**
```
feat(router): add support for catch-all routes
fix(ebpf): resolve memory leak in XDP program
docs(readme): update installation instructions
```

## Testing

### Running Tests

```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage

# Run eBPF tests (requires Linux)
sudo make test-ebpf
```

### Writing Tests

- Follow the AAA pattern: Arrange, Act, Assert
- Use table-driven tests for multiple scenarios
- Ensure new code has test coverage
- Mock external dependencies

## Pull Request Process

1. **Create a branch** from `main` with a descriptive name
2. **Make your changes** following coding standards
3. **Add/update tests** for your changes
4. **Update documentation** if needed
5. **Run tests** locally before submitting
6. **Submit a PR** with a clear description

### PR Checklist

- [ ] Code compiles without errors
- [ ] All tests pass
- [ ] New code has test coverage
- [ ] Documentation updated if needed
- [ ] Commit messages follow conventions
- [ ] PR description explains the change

### Review Process

1. At least one approval required
2. All CI checks must pass
3. No merge conflicts
4. Squash merge to main

## Coding Standards

### Go Code

- Follow [Effective Go](https://golang.org/doc/effective_go)
- Run `gofmt` and `go vet` before committing
- Use `golangci-lint` for comprehensive linting
- Maximum line length: 100 characters
- Functions should be < 50 lines when possible

### eBPF Code

- Use consistent indentation (4 spaces)
- Comment complex logic
- Follow Linux kernel coding style for C code
- Test with multiple kernel versions

### Documentation

- Use Markdown for documentation
- Keep README.md up to date
- Document public APIs with GoDoc comments

## Getting Help

- Open a [Discussion](https://github.com/brattlof/zeptor/discussions) for questions
- Check existing [Issues](https://github.com/brattlof/zeptor/issues) first
- Join our community chat (coming soon)

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
