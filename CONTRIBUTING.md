# Contributing

## Prerequisites

- [Go 1.26+](https://go.dev/dl/)
- [Task](https://taskfile.dev/installation/) (task runner)
- [pre-commit](https://pre-commit.com/#install) (git hooks manager)
- [Docker](https://docs.docker.com/get-docker/) (for integration tests)

## Getting Started

1. Clone the repository and install pre-commit hooks:

```bash
git clone https://github.com/medincident/medincident-zitadel-actions.git
cd medincident-zitadel-actions
pre-commit install --hook-type pre-commit --hook-type pre-push
```

2. Restore Claude Code skills from the lock file:

```bash
npx skills experimental_install
```

3. Verify everything works:

```bash
task test
```

## Pre-commit Hooks

This project uses [pre-commit](https://pre-commit.com/) to run checks before each commit. After installing pre-commit, run:

```bash
pre-commit install --hook-type pre-commit --hook-type pre-push
```

To run all hooks manually against all files:

```bash
pre-commit run --all-files
```

The following hooks are configured:

**On commit:**
- **Whitespace & formatting** — trailing whitespace, end-of-file fixer, YAML validation, large files, merge conflicts
- **Protobuf formatting** — `task fmt:check` (only when `.proto` files change)
- **Lint** — `task lint` (protobuf + Go)

**On push:**
- **Unit tests** — `task test:unit`
- **Vulnerability check** — `task vuln`

## Common Tasks

```bash
task generate          # Generate protobuf and Go enum code
task fmt               # Format protobuf definitions
task fmt:check         # Dry-run format check (no files modified)
task lint              # Lint protobuf and Go code
task lint:proto        # Lint protobuf only
task lint:go           # Lint Go code only
task vuln              # Run Go vulnerability check
task test:unit         # Run unit tests
task test:integration  # Run integration tests (requires Docker)
task test              # Run the full test suite (unit + integration)
```
