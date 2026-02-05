# Claude Code Instructions for pulumi-sendgrid

This file provides guidance for Claude Code when working on this repository.

## Project Overview

This is an autonomous agentic system for building a Pulumi native provider for SendGrid. The system uses specialized agents to implement resources, write tests, run integration tests, and validate changes.

## Guiding Principles

1. **Follow Pulumi Provider Boilerplate Patterns**: Based on https://github.com/pulumi/pulumi-provider-boilerplate
2. **Self-Validating**: Every implementation must pass unit tests, integration tests, and drift tests
3. **State-Driven**: All progress is tracked in `STATE.json` for resumability
4. **Autonomous by Default**: The orchestrator agent drives development without human intervention

## Key Commands

| Command | Description |
|---------|-------------|
| `/start-autonomous` | Begin autonomous development from scratch or resume |
| `/status` | Show current development progress |
| `/implement <resource>` | Implement a specific resource |
| `/test <resource>` | Run all tests for a resource |
| `/verify <resource>` | Verify SendGrid state matches Pulumi state |
| `/resume` | Resume development from last saved state |

## Development Workflow

### Checkpoint After Each Resource

**IMPORTANT:** Commit after completing each resource (all tests passing). This enables:
- Easy rollback if issues arise
- Clear git history showing incremental progress
- Safe resume points for autonomous development

```bash
# After a resource is fully tested:
make codegen
git add .
git commit -m "feat(provider): add {ResourceName} resource

- Implement full CRUD operations
- Add unit tests with mocked HTTP
- Add integration tests with real Pulumi stack
- Verify drift detection and reconciliation"
```

### After Provider Code Changes

**IMPORTANT:** After modifying any Go code in `provider/`, you MUST run `make codegen` before committing.

```bash
# 1. Make changes to provider Go code
# 2. Regenerate schema and SDKs
make codegen

# 3. Commit everything together
git add .
git commit -m "your message"
```

### Key Make Targets

| Command | Description |
|---------|-------------|
| `make codegen` | Regenerate schema + all SDK source files |
| `make build` | Build provider + compile all SDKs |
| `make provider` | Build only the provider binary |
| `make test_provider` | Run provider unit tests |
| `make lint` | Run golangci-lint on provider code |

## Agent Architecture

The system uses specialized agents (in `.claude/agents/`):

- **orchestrator**: Coordinates all work, maintains state
- **schema-expert**: Analyzes OpenAPI specs, designs schemas
- **api-implementer**: Writes Go code for resources
- **unit-tester**: Writes and runs unit tests
- **integration-tester**: Tests against live SendGrid API
- **sendgrid-verifier**: Verifies actual SendGrid state
- **drift-tester**: Tests drift detection and reconciliation
- **pre-commit-validator**: Validates before commits

## Environment Variables

```bash
# Required for integration tests
export SENDGRID_API_KEY="SG.your-api-key-here"

# Optional: Pulumi Cloud backend
export PULUMI_ACCESS_TOKEN="pul-your-token-here"
```

## Project Structure

```
provider/           # Go provider implementation
sdk/                # Generated SDK code (DO NOT edit manually)
tests/
  ├── integration/  # Integration tests with live API
  ├── drift/        # Drift detection tests
  └── e2e/          # End-to-end lifecycle tests
.claude/
  ├── agents/       # Specialized agent definitions
  ├── commands/     # Reusable commands/shortcuts
  └── skills/       # Knowledge and tools
STATE.json          # Development progress state
```

## State Management

All progress is tracked in `STATE.json`. The orchestrator reads this file to:
- Know which resources are implemented
- Track test results
- Resume from failures
- Generate progress reports

## Testing Requirements

Every resource must pass:
1. **Unit Tests**: Mocked HTTP, no API calls (`provider/{resource}_test.go`)
2. **Integration Tests**: Real Pulumi stack with live SendGrid API (`tests/integration/{resource}_stack/`)
3. **Drift Tests**: Out-of-band changes detected and reconciled

### Running Integration Tests

```bash
# Use local Pulumi backend (no cloud account needed)
export PULUMI_BACKEND_URL="file://~/.pulumi-test"
export PULUMI_CONFIG_PASSPHRASE="test"
export SENDGRID_API_KEY="SG.your-key"

# From the test stack directory
cd tests/integration/api_keys_stack
pulumi stack init test
pulumi up --yes      # Create resources
pulumi refresh --yes # Detect drift
pulumi destroy --yes # Clean up
```

## SendGrid API Reference

- Base URL: `https://api.sendgrid.com`
- Auth: `Authorization: Bearer SG.xxxxx`
- OpenAPI specs: https://github.com/twilio/sendgrid-oai
