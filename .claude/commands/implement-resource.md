---
name: implement-resource
description: Implement a single SendGrid API resource. Usage: /implement-resource ApiKey
allowed-tools: Bash, Read, Write, Grep, Glob, Task
---

# Single Resource Implementation

Implement the SendGrid **$ARGUMENTS** resource for the Pulumi provider.

## Step 1: Gather Context

First, read existing implementations to understand the pattern:

```bash
# Read existing resource files as reference
ls provider/*.go
cat provider/provider.go
```

## Step 2: Fetch Schema from SendGrid API Docs

Get the exact request/response schemas from the official SendGrid API documentation:
- https://docs.sendgrid.com/api-reference

## Step 3: Create Implementation Files

Create these files following existing patterns:

### 3.1 API Client: `provider/{resource_lower}.go`

Include:
- Request/Response structs matching SendGrid API JSON
- Validation functions with actionable error messages
- GET, POST, PATCH, DELETE functions with:
  - Context cancellation support
  - Rate limit handling (429) with exponential backoff
  - Proper error handling

### 3.2 Pulumi Resource: `provider/{resource_lower}_resource.go`

Include:
- `{Resource}` controller struct
- `{Resource}Args` input struct with pulumi tags
- `{Resource}State` output struct (embeds Args)
- `Annotate()` methods for descriptions
- `Diff()` - identify replacement vs update
- `Create()` - validate, handle DryRun, call API
- `Read()` - fetch current state
- `Update()` - apply changes
- `Delete()` - remove (404 = success)

### 3.3 Tests: `provider/{resource_lower}_test.go`

Include:
- Validation function tests
- Mock HTTP server tests for each API function
- Error scenario tests (400, 401, 404, 429, 500)

### 3.4 Register in Provider

Add to `provider/provider.go`:
```go
infer.Resource(&{Resource}{}),
```

## Step 4: Verify

```bash
# Build
go build ./provider/...

# Test
go test -v ./provider/... -run {Resource}

# Lint
golangci-lint run ./provider/...
```

## Step 5: Commit

```bash
git add provider/{resource_lower}*.go provider/provider.go
git commit -m "feat({resource_lower}): implement {Resource} resource

- Add {Resource} Pulumi resource with CRUD support
- Add API client for SendGrid {Resource} endpoints
- Add validation and comprehensive error handling
- Add test coverage"
```

## Quality Checklist

Before completing:
- [ ] All inputs validated before API calls
- [ ] Error messages explain what's wrong AND how to fix it
- [ ] Rate limiting handled with exponential backoff
- [ ] Delete is idempotent (404 = success)
- [ ] DryRun returns early without API calls
- [ ] All struct fields have `pulumi:"fieldName"` tags
- [ ] Tests pass
- [ ] Lint passes
