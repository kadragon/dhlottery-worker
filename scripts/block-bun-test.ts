console.error("\n\x1b[31mERROR: This project uses Vitest, not Bun's native test runner.\x1b[0m");
console.error('\x1b[33mUse `bun run test` instead of `bun test`.\x1b[0m\n');
process.exit(1);
