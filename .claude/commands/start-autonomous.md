---
name: start-autonomous
description: Start or resume autonomous provider development. Reads STATE.json and continues from where development left off.
allowed-tools: Bash, Read, Write, Grep, Glob, Task
---

# Start Autonomous Development

Begin or resume the autonomous Pulumi SendGrid provider development process.

## Workflow

### Step 1: Check Current State

Read `STATE.json` to understand current progress:

```bash
cat STATE.json 2>/dev/null || echo "No state file found - starting fresh"
```

### Step 2: Initialize if Fresh Start

If no STATE.json exists, create initial project structure:

1. Create STATE.json with all resources in "pending" state
2. Set up directory structure:
   ```
   mkdir -p provider tests/integration tests/drift examples .claude/agents .claude/commands
   ```
3. Create Makefile, go.mod, and provider scaffolding

### Step 3: Determine Next Task

Based on STATE.json, find the next task:

1. If a resource is "in_progress", continue from that step
2. If no in-progress resource, find first "pending" resource
3. If all resources complete, report success

### Step 4: Execute Task

Invoke the orchestrator agent to manage the development:

```
<uses Task tool to invoke orchestrator agent>
Prompt: "Continue autonomous development. Current state: {STATE.json contents}"
```

### Step 5: Update State

After each significant action:
- Update STATE.json with progress
- Log any errors encountered
- Update metrics

## Initial STATE.json Template

```json
{
  "version": "1.0.0",
  "started_at": "{current_timestamp}",
  "last_updated": "{current_timestamp}",
  "phase": "implementation",
  "resources": {
    "api_keys": {
      "status": "pending",
      "unit_tests": "pending",
      "integration_tests": "pending",
      "drift_tests": "pending"
    },
    "templates": {
      "status": "pending",
      "unit_tests": "pending",
      "integration_tests": "pending",
      "drift_tests": "pending"
    },
    "template_versions": {
      "status": "pending",
      "unit_tests": "pending",
      "integration_tests": "pending",
      "drift_tests": "pending"
    },
    "verified_senders": {
      "status": "pending",
      "unit_tests": "pending",
      "integration_tests": "pending",
      "drift_tests": "pending"
    },
    "domain_authentication": {
      "status": "pending",
      "unit_tests": "pending",
      "integration_tests": "pending",
      "drift_tests": "pending"
    },
    "suppressions": {
      "status": "pending",
      "unit_tests": "pending",
      "integration_tests": "pending",
      "drift_tests": "pending"
    },
    "webhooks": {
      "status": "pending",
      "unit_tests": "pending",
      "integration_tests": "pending",
      "drift_tests": "pending"
    }
  },
  "current_task": null,
  "errors": [],
  "metrics": {
    "resources_completed": 0,
    "resources_total": 7,
    "tests_passed": 0,
    "tests_failed": 0
  }
}
```

## Key Points

1. **Always read STATE.json first** - Never start without understanding current state
2. **Update state after each step** - Enables reliable resumption
3. **Handle errors gracefully** - Log to errors array, continue with next resource
4. **Invoke orchestrator for coordination** - Don't implement resources directly from this command
