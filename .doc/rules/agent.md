# Agent Reading Protocol

Read `.doc/index.json` first and follow `readOrder` progressively.

Read affected module files resolved from `moduleMap` before editing or summarizing module behavior.

When an agent lifecycle hook is installed, let it run `docx finish`; otherwise run `docx sync` before finishing.

`docx finish` is a safe lifecycle-hook wrapper around `docx sync`.

`docx sync` records changed files, updates deterministic module context, and writes an active-agent task under `.doc/tmp/` when semantic follow-up is needed.

Deterministic facts may be refreshed directly; semantic memory requires proposals unless the user confirms the edit.

Use change records for audit trails, module `recentChanges`, proposal evidence, and future AI context.

Prefer finer modules around real workflows when a coarse module hides unrelated concepts.

When active-agent task files already exist under `.doc/tmp/`, read the prompt, produce the requested JSON, then run the matching `docx apply ...` command.

Do not overwrite semantic memory in `.doc/decisions/`, `.doc/mistakes/`, or module `riskRules` without user confirmation. Write proposals instead.
