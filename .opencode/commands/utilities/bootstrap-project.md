---
allowed-tools: Read, Write, Edit, Glob, Grep, Bash(find:*), Bash(rg:*), Bash(git:*), Agent
argument-hint: "[optional-focus]"
description: Bootstrap a new project by generating AGENTS.md and a docs skeleton from repository evidence
---

# Bootstrap Project

Bootstrap this repository into a legible OpenCode project.

Use the current repository state as evidence and generate the minimum project map and docs skeleton needed for an OpenAI-style harness workflow.

<requirements>
- Inspect the current tree, entrypoints, docs, CI, and recent git history
- Generate a concise `AGENTS.md` as a repository map
- Create a minimal `docs/` skeleton if one does not already exist
- Create or refresh workflow docs that explain where research, plans, execution logs, and QA reports live
- Make Chinese the default language for AI chat replies, code comments, and repository documentation unless repository evidence clearly requires another language
- Treat `.opencode/` as assistant tooling unless the target repository explicitly documents it as part of its own developer workflow
- Do not mention OpenCode, `.opencode/`, or other assistant internals in `AGENTS.md` unless the repository itself depends on those surfaces for day-to-day work
- Do not invent architecture, ownership, or commands without evidence
- Prefer short navigational documents over long essays
</requirements>

<actions>
1. Repository discovery:
   - Identify package roots, major directories, and entrypoints
   - Read `README.md`, build files, CI, and existing docs indexes
   - Use current git history to spot renamed or retired paths

2. Generate `AGENTS.md`:
   - Keep it short and navigational
   - Include where to start, what commands to run, and where deeper docs live
   - Point to the canonical docs instead of copying them
   - State that Chinese is the default language for comments, responses, and docs in this repository
   - Omit `.opencode/`, OpenCode, and other assistant-tool implementation details unless the repo already treats them as user-facing workflow surfaces
   - Prefer this compact shape:
     - Project overview
     - Where to start
     - Repository map
     - Commands
     - Sources of truth
     - Change guardrails
     - Validation checklist
   - Keep each section to a few bullets and link outward to docs instead of restating them

3. Generate docs skeleton:
   - Create missing `docs/` folders for design, workflows, decisions, QA, and references as needed
   - Add an index page if the repo lacks one
   - Create a place for research, plan, execution, and QA artifacts
   - Keep `docs/index.md` short and navigational
   - Use `docs/index.md` as the top-level docs map, with deeper pages carrying the durable details

4. Validate:
   - Verify every path you wrote exists
   - Verify every command you documented exists in config or CI
   - Keep the result compact and grounded in evidence
</actions>

<artifact-layout>
- `AGENTS.md`: short repository map and starting points
- `docs/index.md`: navigational index for generated docs
- `docs/workflows/`: research, planning, execution, and QA workflows
- `docs/design/`: architecture notes and design decisions
- `docs/decisions/`: durable decisions and tradeoffs
- `docs/qa/`: validation notes and review output
- `docs/refs/`: source links, evidence, and external references
- `docs/conventions/`: repository-wide conventions such as language and doc style
</artifact-layout>

<output>
Write the generated files directly into the repository and then summarize:
- files created
- files updated
- evidence used
- any unknowns left for human review
</output>
