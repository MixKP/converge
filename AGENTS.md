# AGENTS.md

**Tradeoff:** These guidelines prioritize elegant, scalable architecture, verification, and code quality over raw speed. Do not rush if clarification is needed.

## 1. The Pragmatic Ladder (Think Before Coding)

**Don't assume. Ask if it elevates quality.**

Before writing a single line of code, understand the problem, trace the execution flow end-to-end, and evaluate the request against this ladder. If requirements are ambiguous, edge cases are undefined, or a quick clarification would result in a fundamentally better, more robust solution, **pause and ask**. Do not guess.

Stop at the first rung that satisfies the requirement:

1. **Does this need to be built?** Challenge speculative requirements (YAGNI).
2. **Does it already exist?** Reuse existing helpers, utils, or internal patterns. Do not rewrite them.
3. **Does the standard library cover it?** Lean on native platform capabilities.
4. **Does an installed dependency solve it safely?** Utilize existing ecosystem tools.
5. **Can it be implemented cleanly without new abstractions?** Write the minimum code required.

### Scalability vs. Complexity & Design Patterns

- **Do:** Choose robust algorithms, optimal data structures, and concurrent patterns that naturally scale.
- **Don't:** Confuse scalability with speculative abstraction. Avoid premature factory classes, single-use interfaces, or configuration options that weren't requested.
- **Propose Patterns:** If applying a recognized Design Pattern (e.g., Strategy, Observer, Repository) elegantly solves a structural problem, decouples brittle logic, or significantly improves maintainability, **proactively propose it**. Do not build it silently; use the Question Tool to outline the tradeoff between the added complexity and the architectural benefit.

> **Note on Comments:** Unless explicitly requested, do not include comments in code snippets. The exception is the `tradeoff:` tag: if you must implement a temporary architectural ceiling (e.g., a naive heuristic or a global lock to save time), mark it with a single-line `// tradeoff: [limit] -> [upgrade path]` comment.

## 2. Surgical Changes & Root Causes

**Touch only what you must. Fix the source, not the symptom.**

- **Root Cause Fixes:** A bug report names a symptom. Trace it to its origin. If a shared function is broken, fix it at the source and verify its impact across all callers, rather than patching a single execution path.
- **No Unrelated Polish:** Match the existing style perfectly. Do not "improve" adjacent code, formatting, or comments unless they directly relate to your changes.
- **Orphan Cleanup:** Automatically remove imports, variables, or functions that _your_ changes rendered obsolete. Leave pre-existing dead code alone, but mention it if it blocks clarity.

## 3. Defensive Programming & Edge Cases

**The happy path is not enough. Anticipate failure.**

- **Identify Edge Cases First:** Before implementing logic, explicitly consider null/missing values, type mismatches, empty arrays, network timeouts, and race conditions.
- **Fail Safely:** Never swallow errors silently. Handle them gracefully or bubble them up with clear context.
- **Clarify Unknowns:** If handling a specific edge case significantly complicates the architecture, use the Question Tool to present the tradeoff and ask how to handle it.

## 4. Goal-Driven Execution & Verification

**Define explicit success criteria. Loop until verified.**

Transform every task into verifiable milestones before starting. For multi-step tasks, state a brief execution plan:

    1. [Step/Implementation] → verify: [Specific check/test]
    2. [Step/Implementation] → verify: [Specific check/test]

- **Test Integrity:** Never modify existing tests to make a failing build pass. Fix the implementation to satisfy the assertions. Only modify a test if the requirements have explicitly changed or the test is definitively proven incorrect.
- **Runnable Checks:** Non-trivial logic cannot be left unverified. Leave behind exactly **one** lightweight, runnable check. **This check must verify the happy path AND at least one critical edge case/failure mode.** Do not pull in heavy testing frameworks unless the project already enforces them. Trivial one-liners require no test.

## 5. The Memory Bank (Autonomous Context Management)

**Offload essential knowledge to disk. Do not rely on context windows.**
**CRITICAL:** You MUST perform these steps autonomously in the background. **Do NOT ask the user for permission** to create, read, or update the memory bank.

- **Silent Initialization:** If it does not exist, silently create a `memory/` directory at the project root to act as your external brain.
- **Isolation:** If a `.git` repository exists, silently append `memory/` to `.gitignore`. Do not commit these files.
- **Mandatory Updates:** Whenever you make an architectural decision, complete a milestone, or pause a complex task, you MUST silently write/update `.md` files in this folder (e.g., `memory/architecture.md`, `memory/current_task_state.md`).
- **Recall:** Read from this memory bank automatically to re-orient yourself, prevent hallucinations, and maintain continuity.

## 6. Tool Usage & External Search

**Verify, do not hallucinate.**

- If you are uncertain about a library, framework syntax, or recent API changes, you must not guess.
- Proactively use available MCP tools (such as the DuckDuckGo `ddg` tool) or web search capabilities to look up official documentation and API signatures before writing implementation code.

## 7. OpenCode Question Tool Rules

Use the question tool proactively to propose Design Patterns, when clarification will elevate code quality, prevent brittle implementations, or resolve ambiguous requirements. It is better to ask a clarifying question than to make a poor assumption that degrades the project.

### Configuration Protocol:

- **Header:** Must be <= 12 characters, strictly text-only (no markdown, no punctuation). Treat it as a structural label (e.g., `DataLayout`, `RouteHandling`).
- **Options:** Provide 2–4 distinct options with short labels (1–5 words) accompanied by a concise description of the engineering tradeoffs.
- **Recommendation:** The first option must always be your recommended path and must include the `(Recommended)` suffix.
- **Selection:** Default to `multiple: false` unless a combination of options is structurally valid.
