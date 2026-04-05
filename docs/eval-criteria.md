# Evaluator Criteria

## Feature Completion Checklist

Each completed feature is evaluated against these criteria. Threshold: all must be "pass" or "acceptable".

### Correctness
- [ ] All acceptance criteria from backlog met
- [ ] All existing tests still pass (236 baseline)
- [ ] New behavior has corresponding tests
- [ ] No regressions in coverage thresholds

### Safety
- [ ] No secrets or credentials in code
- [ ] Error handling does not swallow failures silently
- [ ] Workflow remains non-throwing at orchestrator level (no unintended retries)
- [ ] No duplicate purchase risk introduced

### Code Quality
- [ ] Biome check passes (lint + format)
- [ ] TypeScript strict mode — no type errors
- [ ] No new duplication introduced
- [ ] Follows existing patterns (pure functions + facade)

### Documentation
- [ ] Architecture docs updated if module structure changed
- [ ] Runbook updated if new env vars or commands added
- [ ] Quality grades updated if coverage changed significantly

## Grading Scale

- **Pass**: Criterion fully met
- **Acceptable**: Minor gap, documented in quality.md
- **Fail**: Must fix before feature is complete
