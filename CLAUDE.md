# Claude Development Guidelines for MemCP

## Commit Policies
- **Always run tests before committing** - pre-commit hook will automatically run all numbered test files
- **Commit working incremental changes frequently** - don't batch up multiple completed tasks
- **Use descriptive commit messages** with context about what was changed and why
- **Include 🤖 Generated with [Claude Code](https://claude.ai/code) Co-Authored-By: Claude <noreply@anthropic.com>** in commit messages
- **Git commits for every intermediate result that works** - helps track progress and debugging

## Testing Framework
- **All test files must follow numbered format**: `01_basic_sql.yaml`, `02_functions.yaml`, etc.
- **Pre-commit hook scans tests/ directory** and runs all `[0-9][0-9]_*.yaml` files automatically
- **Error tests must use `expect: { error: true }` format** - not `should_fail` or other formats
- **Never use database-specific names** in test files - let test runner use default (`test_db`)
- **Use empty setup/cleanup arrays**: `setup: []` and `cleanup: []` unless specific setup needed
- **Tests that MUST fail** should be in dedicated error test files (like `07_error_cases.yaml`)
- **Single SQL statements per test** - avoid multiple statements with semicolons
- **Run individual tests**: `python3 run_sql_tests.py <file.yaml> 4400`

## Code Standards
- **No comments unless explicitly requested** - keep code clean and self-documenting
- **Follow existing project patterns** - check surrounding code for conventions
- **Check dependencies first** - look at `go.mod`, `package.json` etc. before assuming libraries are available
- **Prefer editing existing files** over creating new ones
- **Never create documentation files** (*.md, README) unless explicitly requested
- **Use TodoWrite tool** for complex multi-step tasks to track progress
- **Mark todos as completed immediately** when tasks are done - don't batch completions

## File Structure
- **tests/**: Numbered YAML test files (`01_*.yaml`, `02_*.yaml`, etc.)
- **run_sql_tests.py**: Test runner that processes YAML specs
- **.git/hooks/pre-commit**: Automatically runs all test files before commits
- **memcp**: Main Go binary, rebuild with `go build -o memcp`

## Development Workflow
1. Use TodoWrite to plan multi-step tasks
2. Make incremental changes and test frequently  
3. Run individual test files during development
4. Commit working intermediate results
5. Pre-commit hook validates all tests automatically
6. Fix any failing tests before final commit

## Testing Commands
```bash
# Run individual test file
python3 run_sql_tests.py tests/01_basic_sql.yaml 4400

# Build memcp
go build -o memcp

# Manual test run (bypasses hook for debugging)
git commit --no-verify -m "message"
```