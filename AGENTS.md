# AGENTS.md

## Test Runner

- Use `bun run test` (Vitest). **Never** `bun test` — Bun's native runner does not support Vitest's `vi.*` API and fails ~60% of this project's tests.
- Guardrail: `bunfig.toml` preloads `scripts/block-bun-test.ts` which aborts `bun test` with a message.

## Operational Log

- TODO: 코드·설정·문서에 직접 드러나지 않는 안정적 운영 제약만 유지.
