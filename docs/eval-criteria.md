# Evaluator Criteria

## Feature Completion Checklist

Each completed feature is evaluated against these criteria. Threshold: all must be "pass" or "acceptable".

### Correctness
- [ ] All acceptance criteria from backlog met
- [ ] All existing tests still pass (`go test ./...`)
- [ ] New behavior has corresponding tests
- [ ] No regression below the 85% statement coverage gate

### Safety
- [ ] No secrets or credentials in code
- [ ] Error handling does not swallow failures silently
- [ ] Workflow remains non-throwing at orchestrator level (no unintended retries)
- [ ] No duplicate purchase risk introduced

### Code Quality
- [ ] `gofmt -l ./cmd ./internal` is empty (formatted)
- [ ] `go vet ./...` passes
- [ ] No new duplication introduced
- [ ] Follows existing patterns (package functions + facade)

### Documentation
- [ ] Architecture docs updated if module structure changed
- [ ] Runbook updated if new env vars or commands added
- [ ] Quality grades updated if coverage changed significantly

## Grading Scale

- **Pass**: Criterion fully met
- **Acceptable**: Minor gap, documented in quality.md
- **Fail**: Must fix before feature is complete
