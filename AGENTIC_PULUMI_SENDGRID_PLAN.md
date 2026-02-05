# Autonomous Agentic Pulumi SendGrid Provider

## Executive Summary

This document outlines a comprehensive plan for building an **autonomous agentic system** that implements a Pulumi provider for SendGrid. The system is designed to be **hands-off** from the start, with specialized agents that can implement resources, write tests, run integration tests against live SendGrid APIs, validate drift detection, and resume development at any point.

The architecture is modeled after the [pulumi-webflow](https://github.com/JDetmar/pulumi-webflow) provider, which uses Claude agents and commands for autonomous development.

---

## Table of Contents

1. [Architecture Overview](#architecture-overview)
2. [Agent Definitions](#agent-definitions)
3. [Skill Definitions](#skill-definitions)
4. [Commands (Shortcuts)](#commands-shortcuts)
5. [SendGrid Resource Mapping](#sendgrid-resource-mapping)
6. [Testing Infrastructure](#testing-infrastructure)
7. [State Management & Resumability](#state-management--resumability)
8. [Development Workflow](#development-workflow)
9. [Directory Structure](#directory-structure)
10. [Getting Started](#getting-started)

---

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                        ORCHESTRATOR AGENT                                    │
│  (Coordinates all work, tracks state, decides next steps)                   │
└─────────────────────────────────────────────────────────────────────────────┘
                                    │
         ┌──────────────────────────┼──────────────────────────┐
         │                          │                          │
         ▼                          ▼                          ▼
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│  IMPLEMENTATION │    │    TESTING      │    │   VALIDATION    │
│     AGENTS      │    │     AGENTS      │    │     AGENTS      │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                          │                          │
    ┌────┴────┐              ┌──────┴──────┐            ┌──────┴──────┐
    │         │              │             │            │             │
    ▼         ▼              ▼             ▼            ▼             ▼
┌───────┐ ┌───────┐    ┌─────────┐ ┌──────────┐  ┌──────────┐ ┌──────────┐
│Schema │ │ API   │    │  Unit   │ │Integration│  │SendGrid  │ │  Drift   │
│Expert │ │Implmtr│    │ Tester  │ │  Tester   │  │Verifier  │ │ Tester   │
└───────┘ └───────┘    └─────────┘ └──────────┘  └──────────┘ └──────────┘
```

### Core Design Principles

1. **Full Autonomy**: The system can start from scratch and implement the entire provider without human intervention
2. **Resumability**: State is persisted so development can be paused and resumed at any point
3. **Self-Validating**: Each agent can validate its own work using tools (tests, API calls, diff checks)
4. **Specialized Agents**: Each agent has a focused responsibility with deep expertise
5. **Progressive Implementation**: Resources are implemented incrementally with full test coverage before moving on

---

## Agent Definitions

### 1. Orchestrator Agent (`orchestrator.md`)

**Purpose**: Coordinates all development work, maintains state, decides what to work on next.

```yaml
---
name: orchestrator
description: |
  Master coordinator for autonomous Pulumi SendGrid provider development.
  Use this agent to:
  - Start or resume provider development
  - Check overall progress and state
  - Decide which resource to implement next
  - Coordinate between specialized agents
model: opus
color: gold
---
```

**Responsibilities**:
- Read/update `STATE.json` to track progress
- Decide which resource to implement next (based on dependencies, complexity)
- Invoke specialized agents in the correct sequence
- Handle failures and retries
- Generate progress reports

**State File Structure** (`STATE.json`):
```json
{
  "version": "1.0.0",
  "started_at": "2026-02-04T10:00:00Z",
  "last_updated": "2026-02-04T15:30:00Z",
  "phase": "implementation",
  "resources": {
    "api_keys": {
      "status": "completed",
      "unit_tests": "passed",
      "integration_tests": "passed",
      "drift_tests": "passed"
    },
    "templates": {
      "status": "in_progress",
      "unit_tests": "passed",
      "integration_tests": "pending",
      "drift_tests": "pending"
    },
    "suppressions": {
      "status": "pending"
    }
  },
  "current_task": {
    "resource": "templates",
    "step": "integration_tests",
    "agent": "integration-tester"
  },
  "errors": [],
  "metrics": {
    "resources_completed": 1,
    "resources_total": 25,
    "tests_passed": 47,
    "tests_failed": 0
  }
}
```

---

### 2. Schema Expert Agent (`schema-expert.md`)

**Purpose**: Analyzes SendGrid OpenAPI specs and designs Pulumi resource schemas.

```yaml
---
name: schema-expert
description: |
  Analyzes SendGrid OpenAPI specifications and designs Pulumi resource schemas.
  Use when:
  - Starting implementation of a new resource
  - Need to understand API structure for a SendGrid endpoint
  - Designing input/output properties for a Pulumi resource
model: opus
color: blue
---
```

**Capabilities**:
- Fetch and parse SendGrid OpenAPI specs from `twilio/sendgrid-oai`
- Map OpenAPI schemas to Go structs
- Design Pulumi resource Args and State structs
- Identify CRUD operation support (which HTTP methods exist)
- Document field validations and constraints

**Key Knowledge**:
```
SendGrid OpenAPI Spec Location: https://github.com/twilio/sendgrid-oai/tree/main/spec/yaml
API Base URL: https://api.sendgrid.com
Auth: Bearer token in Authorization header
```

---

### 3. API Implementer Agent (`api-implementer.md`)

**Purpose**: Implements the Go code for Pulumi resources.

```yaml
---
name: api-implementer
description: |
  Implements Pulumi provider resources for SendGrid APIs in Go.
  Use when:
  - Implementing a new resource after schema design
  - Fixing bugs in existing resource implementations
  - Adding new API methods to existing resources
model: opus
color: green
---
```

**Outputs**:
- `provider/{resource}.go` - API client with HTTP calls
- `provider/{resource}_resource.go` - Pulumi resource implementation
- Updates to `provider/provider.go` to register the resource

**Template Pattern** (from pulumi-webflow):
```go
// provider/template_resource.go
package provider

import (
    "context"
    "fmt"
    p "github.com/pulumi/pulumi-go-provider"
    "github.com/pulumi/pulumi-go-provider/infer"
)

type Template struct{}

type TemplateArgs struct {
    Name       string `pulumi:"name"`
    Generation string `pulumi:"generation,optional"`
}

type TemplateState struct {
    TemplateArgs
    ID        string `pulumi:"id"`
    UpdatedAt string `pulumi:"updatedAt"`
}

func (r *Template) Create(ctx context.Context, req infer.CreateRequest[TemplateArgs]) (infer.CreateResponse[TemplateState], error) {
    // Implementation
}

func (r *Template) Read(ctx context.Context, req infer.ReadRequest[TemplateArgs, TemplateState]) (infer.ReadResponse[TemplateState], error) {
    // Implementation
}

func (r *Template) Update(ctx context.Context, req infer.UpdateRequest[TemplateArgs, TemplateState]) (infer.UpdateResponse[TemplateState], error) {
    // Implementation
}

func (r *Template) Delete(ctx context.Context, req infer.DeleteRequest[TemplateState]) error {
    // Implementation
}
```

---

### 4. Unit Tester Agent (`unit-tester.md`)

**Purpose**: Writes and runs unit tests for provider resources.

```yaml
---
name: unit-tester
description: |
  Writes and runs unit tests for Pulumi provider resources.
  Use when:
  - A new resource implementation is complete
  - Fixing bugs requires new test coverage
  - Need to verify resource behavior without live API calls
model: opus
color: yellow
---
```

**Capabilities**:
- Write Go unit tests with mocked HTTP servers
- Test all CRUD operations
- Test error scenarios (400, 401, 404, 429, 500)
- Test validation functions
- Run tests with `go test -v ./provider/... -run {Resource}`

**Test Pattern**:
```go
func TestCreateTemplate(t *testing.T) {
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Verify request
        if r.Method != "POST" {
            t.Errorf("Expected POST, got %s", r.Method)
        }
        // Return mock response
        w.WriteHeader(201)
        json.NewEncoder(w).Encode(TemplateResponse{ID: "test-id"})
    }))
    defer server.Close()

    // Test implementation
}
```

---

### 5. Integration Tester Agent (`integration-tester.md`)

**Purpose**: Runs integration tests against the live SendGrid API.

```yaml
---
name: integration-tester
description: |
  Runs integration tests against the live SendGrid API using Pulumi stacks.
  Use when:
  - Unit tests pass and need to verify real API behavior
  - Testing CRUD lifecycle with actual SendGrid resources
  - Validating error handling with live API responses

  IMPORTANT: Requires SENDGRID_API_KEY environment variable
model: opus
color: orange
---
```

**Capabilities**:
- Create Pulumi test stacks in `tests/integration/`
- Run `pulumi up` to create real SendGrid resources
- Verify resource creation via SendGrid API
- Run `pulumi destroy` to clean up
- Capture and analyze any failures

**Test Stack Structure**:
```
tests/
├── integration/
│   ├── api_keys/
│   │   ├── Pulumi.yaml
│   │   ├── Pulumi.test.yaml
│   │   └── index.ts
│   ├── templates/
│   │   ├── Pulumi.yaml
│   │   ├── Pulumi.test.yaml
│   │   └── index.ts
│   └── run_integration_tests.sh
```

**Example Test Stack** (`tests/integration/templates/index.ts`):
```typescript
import * as sendgrid from "@yourorg/pulumi-sendgrid";

// Test: Create a template
const template = new sendgrid.Template("test-template", {
    name: `integration-test-${Date.now()}`,
    generation: "dynamic",
});

// Export for verification
export const templateId = template.id;
export const templateName = template.name;
```

---

### 6. SendGrid Verifier Agent (`sendgrid-verifier.md`)

**Purpose**: Verifies that Pulumi operations correctly affected SendGrid state.

```yaml
---
name: sendgrid-verifier
description: |
  Verifies Pulumi operations by directly querying the SendGrid API.
  Use when:
  - Need to confirm a resource was created/updated/deleted in SendGrid
  - Debugging discrepancies between Pulumi state and actual state
  - Validating API response schemas match expectations

  IMPORTANT: Requires SENDGRID_API_KEY environment variable
model: opus
color: cyan
---
```

**Capabilities**:
- Direct HTTP calls to SendGrid API using curl/httpie
- Compare Pulumi state with actual SendGrid state
- Generate verification reports
- Identify state drift

**Verification Script Pattern**:
```bash
#!/bin/bash
# scripts/verify_sendgrid_resource.sh

RESOURCE_TYPE=$1
RESOURCE_ID=$2
API_KEY=${SENDGRID_API_KEY}

case $RESOURCE_TYPE in
  "template")
    curl -s -X GET "https://api.sendgrid.com/v3/templates/${RESOURCE_ID}" \
      -H "Authorization: Bearer ${API_KEY}" \
      -H "Content-Type: application/json"
    ;;
  "api_key")
    curl -s -X GET "https://api.sendgrid.com/v3/api_keys/${RESOURCE_ID}" \
      -H "Authorization: Bearer ${API_KEY}" \
      -H "Content-Type: application/json"
    ;;
esac
```

---

### 7. Drift Tester Agent (`drift-tester.md`)

**Purpose**: Tests drift detection and reconciliation.

```yaml
---
name: drift-tester
description: |
  Tests Pulumi's drift detection by making out-of-band changes to SendGrid
  resources and verifying Pulumi correctly detects and reconciles them.
  Use when:
  - Integration tests pass and need to verify drift handling
  - Testing the Read() implementation for accurate state refresh
  - Validating Update() correctly applies partial changes

  IMPORTANT: Requires SENDGRID_API_KEY environment variable
model: opus
color: red
---
```

**Test Scenarios**:
1. **Create drift**: Create resource via Pulumi, modify via API, run `pulumi refresh`
2. **Delete drift**: Create via Pulumi, delete via API, run `pulumi refresh`
3. **Update reconciliation**: Create, modify via API, run `pulumi up` to reconcile

**Drift Test Script** (`scripts/test_drift.sh`):
```bash
#!/bin/bash
set -e

STACK_NAME="drift-test-$(date +%s)"
RESOURCE_TYPE=$1

echo "=== Drift Test: $RESOURCE_TYPE ==="

# Step 1: Create resource via Pulumi
cd tests/drift/$RESOURCE_TYPE
pulumi stack init $STACK_NAME
pulumi up --yes

# Get resource ID from outputs
RESOURCE_ID=$(pulumi stack output resourceId)

# Step 2: Modify resource directly via SendGrid API
echo "Modifying resource out-of-band..."
./modify_via_api.sh $RESOURCE_ID

# Step 3: Refresh Pulumi state
echo "Running pulumi refresh..."
pulumi refresh --yes --expect-no-changes && {
    echo "ERROR: Drift was not detected!"
    exit 1
}

echo "SUCCESS: Drift was correctly detected"

# Step 4: Reconcile
echo "Running pulumi up to reconcile..."
pulumi up --yes

# Step 5: Verify reconciliation
echo "Verifying reconciliation..."
./verify_reconciliation.sh $RESOURCE_ID

# Cleanup
pulumi destroy --yes
pulumi stack rm $STACK_NAME --yes

echo "=== Drift Test PASSED ==="
```

---

### 8. Pre-Commit Validator Agent (`pre-commit-validator.md`)

**Purpose**: Validates code before committing (runs codegen, build, lint, tests).

```yaml
---
name: pre-commit-validator
description: |
  Validates all code changes before committing to ensure CI will pass.
  Use when:
  - Ready to commit changes
  - Preparing a pull request
  - Need to verify the build is in a good state
model: opus
color: purple
---
```

**Validation Steps**:
1. Run `make codegen` (regenerate schema + SDKs)
2. Verify worktree is clean (no unexpected changes)
3. Run `make build` (compile provider + SDKs)
4. Run `make lint` (golangci-lint)
5. Run `make test_provider` (unit tests)

---

## Skill Definitions

Skills provide reusable knowledge and tools for agents.

### 1. SendGrid Docs Skill (`sendgrid-docs/`)

**Purpose**: Fetches and searches SendGrid API documentation.

```yaml
---
name: sendgrid-docs
description: |
  Fetches SendGrid API documentation and OpenAPI specifications.
  Use when implementing resources to get accurate API schemas.
---
```

**Contents**:
- `SKILL.md` - Instructions for fetching docs
- `scripts/fetch_openapi.py` - Download OpenAPI spec for a resource
- `references/api_overview.md` - SendGrid API conventions

### 2. Pulumi Provider Patterns Skill (`pulumi-patterns/`)

**Purpose**: Reference patterns for Pulumi provider development.

```yaml
---
name: pulumi-patterns
description: |
  Reference patterns and best practices for Pulumi native provider development.
  Use when implementing resources to ensure consistency.
---
```

**Contents**:
- `SKILL.md` - Core patterns overview
- `references/crud_patterns.md` - CRUD implementation patterns
- `references/error_handling.md` - Error handling patterns
- `references/testing_patterns.md` - Testing patterns

### 3. Integration Test Skill (`integration-tests/`)

**Purpose**: Tools and patterns for integration testing.

```yaml
---
name: integration-tests
description: |
  Tools and patterns for running Pulumi integration tests against SendGrid.
  Use when setting up or running integration test suites.
---
```

**Contents**:
- `SKILL.md` - Integration testing workflow
- `scripts/setup_test_stack.sh` - Initialize a test stack
- `scripts/run_tests.sh` - Run integration tests
- `assets/test_stack_template/` - Template for test stacks

---

## Commands (Shortcuts)

Commands are quick actions that can be invoked with `/command-name`.

### `/implement <resource>`

Implements a single SendGrid resource end-to-end.

```yaml
---
name: implement
description: Implement a SendGrid resource. Usage: /implement templates
allowed-tools: Bash, Read, Write, Grep, Glob, Task
---
```

**Workflow**:
1. Fetch OpenAPI spec for the resource
2. Design schema (Args, State structs)
3. Implement API client
4. Implement Pulumi resource
5. Write unit tests
6. Run unit tests
7. Update STATE.json

### `/test <resource>`

Runs all tests for a resource.

```yaml
---
name: test
description: Run all tests for a resource. Usage: /test templates
allowed-tools: Bash, Read, Task
---
```

**Workflow**:
1. Run unit tests
2. Run integration tests
3. Run drift tests
4. Update STATE.json with results

### `/verify <resource>`

Verifies SendGrid state matches Pulumi state.

```yaml
---
name: verify
description: Verify SendGrid state for a resource. Usage: /verify templates
allowed-tools: Bash, Read, Task
---
```

### `/status`

Shows current development progress.

```yaml
---
name: status
description: Show current development progress and state.
allowed-tools: Read
---
```

### `/resume`

Resumes development from the last saved state.

```yaml
---
name: resume
description: Resume autonomous development from the last saved state.
allowed-tools: Bash, Read, Write, Task
---
```

---

## SendGrid Resource Mapping

Based on the SendGrid OpenAPI specification, here are the resources to implement:

### Phase 1: Core Resources (MVP)

| Resource | OpenAPI Spec | CRUD Support | Priority |
|----------|--------------|--------------|----------|
| `ApiKey` | `tsg_api_keys_v3.yaml` | Create, Read, Update, Delete | P0 |
| `Template` | `tsg_templates_v3.yaml` | Create, Read, Update, Delete | P0 |
| `TemplateVersion` | `tsg_templates_v3.yaml` | Create, Read, Update, Delete | P0 |

### Phase 2: Email Infrastructure

| Resource | OpenAPI Spec | CRUD Support | Priority |
|----------|--------------|--------------|----------|
| `VerifiedSender` | `tsg_verified_senders_v3.yaml` | Create, Read, Delete | P1 |
| `DomainAuthentication` | `tsg_domain_authentication_v3.yaml` | Create, Read, Update, Delete | P1 |
| `LinkBranding` | `tsg_link_branding_v3.yaml` | Create, Read, Update, Delete | P1 |
| `IpPool` | `tsg_ips_v3.yaml` | Create, Read, Update, Delete | P1 |

### Phase 3: Suppressions & Compliance

| Resource | OpenAPI Spec | CRUD Support | Priority |
|----------|--------------|--------------|----------|
| `UnsubscribeGroup` | `tsg_suppressions_v3.yaml` | Create, Read, Update, Delete | P2 |
| `GlobalSuppression` | `tsg_suppressions_v3.yaml` | Create, Read, Delete | P2 |
| `BounceSupprression` | `tsg_suppressions_v3.yaml` | Create, Read, Delete | P2 |

### Phase 4: Webhooks & Integrations

| Resource | OpenAPI Spec | CRUD Support | Priority |
|----------|--------------|--------------|----------|
| `Webhook` | `tsg_webhooks_v3.yaml` | Create, Read, Update, Delete | P2 |
| `ParseSetting` | `tsg_inbound_parse_v3.yaml` | Create, Read, Update, Delete | P2 |

### Phase 5: Marketing Automation

| Resource | OpenAPI Spec | CRUD Support | Priority |
|----------|--------------|--------------|----------|
| `Contact` | `tsg_mc_contacts_v3.yaml` | Create, Read, Update, Delete | P3 |
| `ContactList` | `tsg_mc_lists_v3.yaml` | Create, Read, Update, Delete | P3 |
| `Segment` | `tsg_mc_segments_v3.yaml` | Create, Read, Update, Delete | P3 |
| `SingleSend` | `tsg_mc_singlesends_v3.yaml` | Create, Read, Update, Delete | P3 |

### Phase 6: Users & Access

| Resource | OpenAPI Spec | CRUD Support | Priority |
|----------|--------------|--------------|----------|
| `Subuser` | `tsg_subusers_v3.yaml` | Create, Read, Update, Delete | P3 |
| `Teammate` | `tsg_teammates_v3.yaml` | Create, Read, Update, Delete | P3 |
| `Alert` | `tsg_alerts_v3.yaml` | Create, Read, Update, Delete | P3 |

---

## Testing Infrastructure

### Directory Structure

```
tests/
├── unit/                          # Go unit tests (in provider/)
├── integration/
│   ├── Makefile
│   ├── setup.sh                   # Initialize test environment
│   ├── api_keys/
│   │   ├── Pulumi.yaml
│   │   ├── Pulumi.test.yaml
│   │   └── index.ts
│   ├── templates/
│   │   └── ...
│   └── run_all.sh
├── drift/
│   ├── Makefile
│   ├── api_keys/
│   │   ├── Pulumi.yaml
│   │   ├── modify_via_api.sh      # Makes out-of-band change
│   │   ├── verify_reconciliation.sh
│   │   └── index.ts
│   └── templates/
│       └── ...
└── e2e/
    ├── Makefile
    └── full_lifecycle_test.sh     # Complete create/update/delete cycle
```

### Test Environment Setup

```bash
#!/bin/bash
# scripts/setup_test_environment.sh

# Required environment variables
export SENDGRID_API_KEY="${SENDGRID_API_KEY:?'SENDGRID_API_KEY is required'}"
export PULUMI_ACCESS_TOKEN="${PULUMI_ACCESS_TOKEN:-}"  # Optional, uses local backend if not set

# Use local Pulumi backend for testing
if [ -z "$PULUMI_ACCESS_TOKEN" ]; then
    export PULUMI_BACKEND_URL="file://~/.pulumi-test"
fi

# Install provider locally
make provider
install -m 755 bin/pulumi-resource-sendgrid ~/.pulumi/plugins/resource-sendgrid-v0.0.1/

# Verify
pulumi plugin ls | grep sendgrid
```

### Integration Test Runner

```bash
#!/bin/bash
# tests/integration/run_all.sh

set -e

RESOURCES=("api_keys" "templates" "verified_senders")
FAILED=()

for resource in "${RESOURCES[@]}"; do
    echo "=== Testing $resource ==="
    cd $resource

    STACK_NAME="test-$resource-$(date +%s)"
    pulumi stack init $STACK_NAME --secrets-provider passphrase

    if pulumi up --yes; then
        echo "✓ Create succeeded"

        # Verify via API
        if ./verify.sh; then
            echo "✓ Verification passed"
        else
            echo "✗ Verification failed"
            FAILED+=($resource)
        fi

        # Cleanup
        pulumi destroy --yes
        pulumi stack rm $STACK_NAME --yes
    else
        echo "✗ Create failed"
        FAILED+=($resource)
    fi

    cd ..
done

if [ ${#FAILED[@]} -gt 0 ]; then
    echo "FAILED: ${FAILED[@]}"
    exit 1
fi

echo "All integration tests passed!"
```

---

## State Management & Resumability

### State File (`STATE.json`)

The orchestrator agent maintains a state file that tracks:
- Overall progress
- Status of each resource (pending, in_progress, completed, failed)
- Test results for each resource
- Current task being worked on
- Error history

### State Transitions

```
pending → implementing → unit_testing → integration_testing → drift_testing → completed
                ↓              ↓                ↓                   ↓
              failed         failed           failed              failed
```

### Resume Logic

When `/resume` is invoked:

1. Read `STATE.json`
2. Find current task or next pending resource
3. Continue from the appropriate step:
   - If `implementing`: Continue implementation
   - If `unit_testing`: Run unit tests
   - If `integration_testing`: Run integration tests
   - If `drift_testing`: Run drift tests
   - If `failed`: Analyze error and retry or escalate

### Checkpointing

After each significant step, the state is saved:

```python
# Pseudocode for state updates
def complete_step(resource, step, result):
    state = read_state()
    state['resources'][resource][step] = result
    state['last_updated'] = now()

    if result == 'failed':
        state['errors'].append({
            'resource': resource,
            'step': step,
            'error': get_last_error(),
            'timestamp': now()
        })

    if all_steps_passed(resource):
        state['resources'][resource]['status'] = 'completed'
        state['metrics']['resources_completed'] += 1

    write_state(state)
```

---

## Development Workflow

### Autonomous Start

When starting fresh development:

```bash
# Initialize the project
claude "/init-provider sendgrid"

# Start autonomous development
claude "/start-autonomous"
```

The orchestrator will:
1. Initialize `STATE.json`
2. Set up project structure
3. Begin implementing Phase 1 resources
4. Continue until complete or stopped

### Manual Intervention

If needed, you can:
- Check status: `/status`
- Implement a specific resource: `/implement templates`
- Run tests: `/test templates`
- Resume after pause: `/resume`

### Typical Session Flow

```
1. Orchestrator reads STATE.json
2. Identifies next resource to implement (e.g., templates)
3. Invokes schema-expert to analyze OpenAPI spec
4. Invokes api-implementer to write code
5. Invokes unit-tester to write and run tests
6. If tests pass, invokes integration-tester
7. If integration passes, invokes drift-tester
8. Updates STATE.json with results
9. Moves to next resource
10. Repeats until all resources complete
```

---

## Directory Structure

```
pulumi-sendgrid/
├── .claude/
│   ├── agents/
│   │   ├── orchestrator.md
│   │   ├── schema-expert.md
│   │   ├── api-implementer.md
│   │   ├── unit-tester.md
│   │   ├── integration-tester.md
│   │   ├── sendgrid-verifier.md
│   │   ├── drift-tester.md
│   │   └── pre-commit-validator.md
│   ├── commands/
│   │   ├── implement.md
│   │   ├── test.md
│   │   ├── verify.md
│   │   ├── status.md
│   │   └── resume.md
│   └── skills/
│       ├── sendgrid-docs/
│       │   ├── SKILL.md
│       │   └── scripts/fetch_openapi.py
│       ├── pulumi-patterns/
│       │   ├── SKILL.md
│       │   └── references/
│       └── integration-tests/
│           ├── SKILL.md
│           ├── scripts/
│           └── assets/
├── provider/
│   ├── provider.go
│   ├── config.go
│   ├── api_key.go
│   ├── api_key_resource.go
│   ├── api_key_test.go
│   ├── template.go
│   ├── template_resource.go
│   ├── template_test.go
│   └── cmd/
│       └── pulumi-resource-sendgrid/
│           ├── main.go
│           └── schema.json
├── sdk/
│   ├── go/
│   ├── nodejs/
│   ├── python/
│   ├── dotnet/
│   └── java/
├── tests/
│   ├── integration/
│   ├── drift/
│   └── e2e/
├── scripts/
│   ├── setup_test_environment.sh
│   ├── verify_sendgrid_resource.sh
│   └── test_drift.sh
├── examples/
│   ├── api_keys/
│   │   └── typescript/
│   └── templates/
│       └── typescript/
├── STATE.json
├── CLAUDE.md
├── Makefile
├── go.mod
├── go.sum
└── README.md
```

---

## Getting Started

### Prerequisites

1. **Go 1.21+**: For provider development
2. **Node.js 18+**: For TypeScript SDK and examples
3. **Pulumi CLI 3.0+**: For testing
4. **SendGrid Account**: For integration testing

### SendGrid Account Setup

1. Create a SendGrid account at https://sendgrid.com
2. Navigate to Settings → API Keys
3. Create an API key with full access (for development)
4. Save the key securely - you'll need it for `SENDGRID_API_KEY`

### Environment Setup

```bash
# Required for integration tests
export SENDGRID_API_KEY="SG.your-api-key-here"

# Optional: Use Pulumi Cloud backend
export PULUMI_ACCESS_TOKEN="pul-your-token-here"

# Or use local backend (default if no token)
export PULUMI_BACKEND_URL="file://~/.pulumi"
```

### Initialize the Project

```bash
# Clone or create the repo
mkdir pulumi-sendgrid && cd pulumi-sendgrid

# Initialize with Claude
claude "Initialize the pulumi-sendgrid provider project structure"

# Start autonomous development
claude "/start-autonomous"
```

### Manual Development

If you prefer step-by-step:

```bash
# Implement a specific resource
claude "/implement api_keys"

# Run tests
claude "/test api_keys"

# Check status
claude "/status"
```

---

## Appendix A: SendGrid API Quick Reference

### Authentication

All requests require:
```
Authorization: Bearer SG.xxxxx
Content-Type: application/json
```

### Base URLs

- Global: `https://api.sendgrid.com`
- EU: `https://api.eu.sendgrid.com`

### Common Response Codes

| Code | Meaning |
|------|---------|
| 200 | Success |
| 201 | Created |
| 204 | Deleted (no content) |
| 400 | Bad request |
| 401 | Unauthorized |
| 403 | Forbidden |
| 404 | Not found |
| 429 | Rate limited |
| 500 | Server error |

### Rate Limiting

SendGrid rate limits vary by endpoint. Handle 429 responses with exponential backoff:
- Wait 1s, 2s, 4s, 8s... up to 32s
- Max 5 retries

---

## Appendix B: Agent Communication Protocol

Agents communicate through:

1. **STATE.json**: Shared state file
2. **Task outputs**: Return values from agent invocations
3. **File system**: Generated code and test results

### Invoking Sub-Agents

```markdown
<!-- In orchestrator.md -->

When implementing a resource:

1. Invoke schema-expert:
   ```
   <uses Task tool to invoke schema-expert>
   Prompt: "Analyze the SendGrid OpenAPI spec for the {resource} endpoint and design the Pulumi resource schema."
   ```

2. Wait for schema design, then invoke api-implementer:
   ```
   <uses Task tool to invoke api-implementer>
   Prompt: "Implement the {resource} Pulumi resource using this schema design: {schema_output}"
   ```
```

---

## Appendix C: Example Resource Implementation

### API Keys Resource (Complete Example)

**Step 1: Schema Design** (from schema-expert)

```
Resource: ApiKey
OpenAPI: tsg_api_keys_v3.yaml

Input Properties (Args):
- name: string (required) - Name of the API key
- scopes: []string (optional) - Permission scopes

Output Properties (State):
- apiKeyId: string - SendGrid's internal ID
- apiKey: string (secret) - The actual key (only returned on create)

CRUD Support:
- Create: POST /v3/api_keys
- Read: GET /v3/api_keys/{api_key_id}
- Update: PATCH /v3/api_keys/{api_key_id} (name only)
         PUT /v3/api_keys/{api_key_id} (name + scopes)
- Delete: DELETE /v3/api_keys/{api_key_id}
```

**Step 2: Implementation** (from api-implementer)

See `provider/api_key.go` and `provider/api_key_resource.go` templates above.

**Step 3: Unit Tests** (from unit-tester)

```go
func TestCreateApiKey(t *testing.T) {
    // Test implementation
}

func TestReadApiKey(t *testing.T) {
    // Test implementation
}
```

**Step 4: Integration Test** (from integration-tester)

```typescript
// tests/integration/api_keys/index.ts
import * as sendgrid from "@yourorg/pulumi-sendgrid";

const apiKey = new sendgrid.ApiKey("test-key", {
    name: `pulumi-test-${Date.now()}`,
    scopes: ["mail.send"],
});

export const apiKeyId = apiKey.apiKeyId;
```

**Step 5: Drift Test** (from drift-tester)

```bash
# tests/drift/api_keys/modify_via_api.sh
#!/bin/bash
API_KEY_ID=$1
curl -X PATCH "https://api.sendgrid.com/v3/api_keys/${API_KEY_ID}" \
  -H "Authorization: Bearer ${SENDGRID_API_KEY}" \
  -H "Content-Type: application/json" \
  -d '{"name": "modified-out-of-band"}'
```

---

*This plan provides a complete blueprint for autonomous Pulumi provider development. The system is designed to be self-sustaining once started, with full test coverage and validation at each step.*
