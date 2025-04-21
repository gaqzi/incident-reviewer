# Incident Reviewer Development Guide

## Commands
- **Setup**: `script/bootstrap` - Install dependencies
- **Run**: `script/server` - Start development server
- **Test**: `go test -race ./...` - Run all tests
- **Single test**: `go test -v ./path/to/package -run TestName`
- **Package tests**: `go test -v ./path/to/package`
- **Lint**: `golangci-lint run`
- **Tech debt**: `script/techdebt` - Find tech debt comments

## Code Style & Structure
- **Architecture**: Clean architecture with domain-driven design
- **Organization**: cmd (entry points), internal (app logic, domain models)
- **Error handling**: Domain-specific errors, error wrapping with context
- **Naming**: Interfaces end with 'er' (e.g., Mapper), domain storage follows [Domain]Storage pattern
- **Testing**:
    - Use `stretchr/testify/require` with direct calls (`require.X(t, ...)`)
    - Use "expected" instead of "should" in assertion messages (e.g., "expected error when file not found")
    - Assert wrapped error message using `require.Contains(t, "substrings", err.Error())`
    - Write imperative, action-oriented test names (e.g., "Return Error When File Not Found")
    - Structure using independent `t.Run()` cases following Arrange/Act/Assert pattern (no comments)
    - Test behavior and outcomes, not implementation details
    - Keep tests focused on functionality, not internal state
    - Use test doubles (like `fstest.MapFS` and mocks from `stretchr/testify/mock`) to mock dependencies
    - Use setup helpers and `t.Cleanup` for test independence
- **Web**: HTML templates with htmx for dynamic updates
- **Linting**: Custom ruleguard rules (don't pass domain objects to templates)
- **Database**: PostgreSQL with goose migrations

## Domain Knowledge
- The app deals with incident reviews and contributing causes
- Models in `internal/normalized` and `internal/reviewing`
- Web handlers in `internal/app/web`

## Pattern definitions as prompts
- Look in `docs/ai/test-builder-pattern.md` for the definition of "test builders"
