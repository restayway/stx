# Contributing to STX

Thank you for your interest in contributing to STX! This document provides guidelines for contributors.

## Development Setup

1. **Fork the repository** on GitHub
2. **Clone your fork** locally:
   ```bash
   git clone https://github.com/yourusername/stx.git
   cd stx
   ```
3. **Install dependencies**:
   ```bash
   make deps
   ```

## Development Workflow

1. **Create a feature branch**:
   ```bash
   git checkout -b feature/your-feature-name
   ```

2. **Make your changes** and ensure they include:
   - Clear, well-documented code
   - Comprehensive test coverage
   - Updated documentation if needed

3. **Run the test suite**:
   ```bash
   make check
   ```

4. **Commit your changes**:
   ```bash
   git commit -m "Add feature: your feature description"
   ```

5. **Push to your fork**:
   ```bash
   git push origin feature/your-feature-name
   ```

6. **Create a Pull Request** on GitHub

## Code Standards

- **Go version**: Go 1.21 or higher
- **Testing**: Maintain 100% test coverage
- **Code style**: Follow Go conventions and use `gofmt`
- **Linting**: Code must pass `golangci-lint`
- **Documentation**: All public functions must have Go doc comments

## Testing

All contributions must include comprehensive tests:

```bash
# Run tests
make test

# Run tests with coverage
make cover

# Run all checks (formatting, linting, testing)
make check
```

## Pull Request Guidelines

- **Title**: Clear, descriptive title
- **Description**: Explain what your PR does and why
- **Tests**: All tests must pass
- **Coverage**: Maintain 100% test coverage
- **Documentation**: Update docs if needed

## Code Review Process

1. All PRs require at least one review from a maintainer
2. CI must pass (tests, linting, coverage)
3. No merge conflicts
4. All feedback must be addressed

## Issue Guidelines

When opening an issue, please provide:
- Clear description of the problem or feature request
- Steps to reproduce (for bugs)
- Expected vs actual behavior
- Go version and environment details

## Getting Help

- Open an issue for bugs or feature requests
- Check existing issues before creating new ones
- Be respectful and constructive in discussions

## License

By contributing to STX, you agree that your contributions will be licensed under the MIT License.