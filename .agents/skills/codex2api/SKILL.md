```markdown
# codex2api Development Patterns

> Auto-generated skill from repository analysis

## Overview
This skill teaches you the core development patterns and conventions used in the `codex2api` Go codebase. You'll learn about file organization, import/export styles, commit conventions, and how to write and run tests. While no explicit workflows were detected, this guide provides best practices and suggested commands for common tasks in this repository.

## Coding Conventions

### File Naming
- Use **snake_case** for all file names.
  - Example: `api_handler.go`, `user_service.go`

### Import Style
- Use **relative imports** for internal packages.
  - Example:
    ```go
    import (
        "fmt"
        "../utils"
    )
    ```

### Export Style
- Use **named exports** for functions, types, and variables.
  - Example:
    ```go
    // Exported function
    func GetUser(id int) (*User, error) {
        // ...
    }

    // Exported type
    type User struct {
        ID   int
        Name string
    }
    ```

### Commit Messages
- Follow the **conventional commit** format.
- Use prefixes like `fix` or `chore`.
  - Example:
    ```
    fix: correct user ID validation logic
    chore: update dependencies for security
    ```

## Workflows

### Creating a New Feature
**Trigger:** When adding new functionality to the codebase  
**Command:** `/new-feature`

1. Create a new file using snake_case (e.g., `new_feature.go`).
2. Write your code, using named exports for any public functions or types.
3. Use relative imports for any internal dependencies.
4. Write corresponding tests in a file matching `*.test.*` (e.g., `new_feature.test.go`).
5. Commit your changes using a conventional commit message, e.g., `feat: add new feature for X`.

### Fixing a Bug
**Trigger:** When resolving a bug or issue  
**Command:** `/fix-bug`

1. Identify the bug and the affected files.
2. Make the necessary code changes.
3. Add or update tests in the corresponding `*.test.*` file to cover the fix.
4. Commit using a `fix:` prefix, e.g., `fix: resolve panic on nil pointer`.

### Running Tests
**Trigger:** Before pushing or merging code  
**Command:** `/run-tests`

1. Locate all test files matching `*.test.*`.
2. Use Go's testing tools to run the tests:
    ```sh
    go test ./...
    ```
3. Ensure all tests pass before proceeding.

## Testing Patterns

- Test files follow the pattern `*.test.*` (e.g., `api_handler.test.go`).
- Use Go's built-in testing framework (assumed, as no explicit framework detected).
- Example test file:
    ```go
    package main

    import "testing"

    func TestGetUser(t *testing.T) {
        user, err := GetUser(1)
        if err != nil {
            t.Fatalf("expected no error, got %v", err)
        }
        if user.ID != 1 {
            t.Errorf("expected ID 1, got %d", user.ID)
        }
    }
    ```

## Commands
| Command        | Purpose                                         |
|----------------|-------------------------------------------------|
| /new-feature   | Scaffold and implement a new feature            |
| /fix-bug       | Apply and commit a bug fix                      |
| /run-tests     | Run all tests in the repository                 |
```
