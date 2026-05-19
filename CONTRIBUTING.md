# Contributing

Thank you for your interest in contributing to this project!

## Development Environment Setup

### Prerequisites

- Docker Desktop (or Docker Engine)
- Go 1.23+
- Python 3.12+
- [uv](https://github.com/astral-sh/uv) (fast Python package manager)
- [pre-commit](https://pre-commit.com/)

### Setup Steps

1. Clone the repository:
   ```bash
   git clone https://github.com/seri114/docker-sandbox.git
   cd docker-sandbox
   ```

2. Install pre-commit hooks:
   ```bash
   uv pip install pre-commit
   pre-commit install
   ```

3. Install Python dependencies:
   ```bash
   cd test-client
   uv sync
   ```

4. Start the development environment:
   ```bash
   docker compose up --build
   ```

## Running Tests

### Go Tests (sandbox-controller)

```bash
cd sandbox-controller

# Unit tests
go test ./...

# Integration tests (requires Docker)
go test -v ./... -run Integration

# E2E tests (requires Docker)
go test -v ./... -run E2E
```

### Python Tests (test-client)

```bash
cd test-client

# Install test dependencies
uv pip install pytest pytest-asyncio httpx

# Unit tests
pytest

# E2E tests (requires controller running)
pytest tests/test_e2e.py -v
```

## Code Style

### Go

- Use `gofmt` for formatting
- Run `go vet` for linting
- Follow [Effective Go](https://go.dev/doc/effective_go) guidelines

```bash
cd sandbox-controller
gofmt -w .
go vet ./...
```

### Python

- Use `ruff` for formatting and linting
- Follow PEP 8 guidelines

```bash
cd test-client
ruff check .
ruff format .
```

## Submitting Changes

1. Create a new branch for your work:
   ```bash
   git checkout -b feature/your-feature-name
   ```

2. Make your changes and ensure tests pass:
   ```bash
   # Go tests
   cd sandbox-controller && go test ./...

   # Python tests
   cd test-client && pytest

   # Pre-commit checks
   pre-commit run --all-files
   ```

3. Commit your changes:
   ```bash
   git add .
   git commit -m "feat: add your feature description"
   ```

4. Push to your fork:
   ```bash
   git push origin feature/your-feature-name
   ```

5. Open a pull request

## Pull Request Guidelines

- Provide a clear description of the changes
- Reference related issues (e.g., `Fixes #123`)
- Ensure all tests pass
- Add tests for new features
- Update documentation as needed
- Keep changes focused and atomic

## Pre-commit Hooks

This project uses pre-commit to automatically check code quality:

- **Go**: `gofmt`, `go vet`
- **Python**: `ruff` (format + lint)
- **General**: trailing whitespace, final newline, large file warnings

### Running Pre-commit Manually

```bash
# Run all checks
pre-commit run --all-files

# Skip for a specific commit
git commit --no-verify
```

## Getting Help

If you have questions:
- Open a GitHub Discussion
- Create an issue with the `question` label
- Check existing documentation

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
