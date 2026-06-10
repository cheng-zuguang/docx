<!-- docx:start -->
## Project Context

Before work, read `.doc/index.json`.

Follow its `readOrder` progressively. Resolve edited paths with `moduleMap`; inspect decisions and recent changes for behavior changes; inspect mistakes while debugging or reviewing.

Keep `AGENTS.md` short; use `.doc/rules/agent.md` for detailed behavior.

When an agent lifecycle hook is installed, let it run `docx finish`; otherwise run `docx sync` before finishing.

Use change records for audit trails, module `recentChanges`, proposal evidence, and future AI context.

Do not overwrite semantic memory in `.doc/decisions/`, `.doc/mistakes/`, or module `riskRules` without user confirmation. Write proposals instead.
<!-- docx:end -->
