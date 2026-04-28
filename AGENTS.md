# AGENTS.md

## Test Runner

- Use `bun run test` (Vitest). **Never** `bun test` — Bun's native runner does not support Vitest's `vi.*` API and fails ~60% of this project's tests.
- Guardrail: `bunfig.toml` preloads `scripts/block-bun-test.ts` which aborts `bun test` when invoked from the repo root. Bun does not walk up for `bunfig.toml`, so running `bun test` from a subdirectory bypasses the guard — always invoke from the repo root.

> 운영 지식 (엔드포인트, 비즈니스 규칙, 디버깅 명령, 핵심 결정 이력)은 `docs/` 참조.
