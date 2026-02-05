---
name: status
description: Show current development progress and state for the Pulumi SendGrid provider.
allowed-tools: Read
---

# Show Development Status

Display the current state of the autonomous Pulumi SendGrid provider development.

## Workflow

### Step 1: Read State

```bash
cat STATE.json
```

### Step 2: Generate Report

Parse STATE.json and output a formatted status report:

```markdown
## SendGrid Provider Development Status

**Phase**: {phase}
**Progress**: {completed}/{total} resources ({percentage}%)
**Last Updated**: {last_updated}

### Completed Resources
{list resources with status="completed" and their test results}

### In Progress
{list resources with status other than "pending" or "completed"}

### Pending
{list resources with status="pending"}

### Failed (Needs Attention)
{list resources with any failed tests, include error messages}

### Metrics
- Total Resources: {total}
- Completed: {completed}
- Unit Tests Passed: {unit_passed}
- Integration Tests Passed: {integration_passed}
- Drift Tests Passed: {drift_passed}
- Tests Failed: {tests_failed}

### Current Task
{if current_task exists, show what's being worked on}

### Recent Errors
{list last 5 errors from errors array, if any}
```

### Step 3: Recommendations

Based on state, suggest next actions:

- If resources are pending: "Run `/start-autonomous` to continue development"
- If resources failed: "Run `/implement {resource}` to retry failed resource"
- If all complete: "Provider development complete! Ready for release."
