# Quality Grades

Go port snapshot. Numbers from `go test ./... -cover` + `golangci-lint run ./...`.

## Coverage

- **Total**: 86.7% of statements (gate ≥85%, CI-enforced)
- **Tool**: `go test ./... -coverprofile=coverage.out && go tool cover -func=coverage.out`

| Package | Coverage | Notes |
|---------|----------|-------|
| `internal/datekst` | 100.0% | |
| `internal/dherr` | 100.0% | |
| `internal/env` | 100.0% | |
| `internal/format` | 100.0% | |
| `internal/workflow` | 100.0% | |
| `internal/notify` | 93.1% | |
| `internal/testutil` | 94.7% | test-support library |
| `internal/logger` | 97.1% | |
| `internal/httpclient` | 86.0% | |
| `internal/dhlottery` | 84.7% | Largest package |
| `cmd/worker` | 83.3% | |
| `cmd/realtest` | 40.0% | Live-endpoint harness — hard to unit-test by design |
| `internal/constants` | n/a | Constants only; no test file |

## Lint

`golangci-lint run ./...` → **0 issues**

Enabled linters (`.golangci.yml`): staticcheck, errcheck, bodyclose, gosec (G404 excluded),
predeclared, noctx, exhaustive, unparam, goconst.

## Known Gaps

1. `cmd/realtest` low coverage — live-endpoint harness; not expected to be fully unit-testable.
2. `internal/constants` has no test file — constants only, low risk.
3. `internal/dhlottery/pension_reserve.go` is the largest source file; candidate for decomposition
   if complexity grows.
