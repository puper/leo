---
description: Perform a comprehensive audit of existing code (Logic, Security, Performance) and generate reproduction unit tests. [MAPPED TO /audit]
allowed-tools:
  - Read
  - Grep
  - Glob
  - Bash
  - Task
  - Write
  - Edit
argument-hint: "[path/to/file_or_dir]"
---

# /audit - Comprehensive Code Audit

Use this command to perform a high-precision audit for **Logic Bugs, Security Risks, and Performance Issues**. This command orchestrates a multi-agent review and automatically generates unit tests for its findings.

## 1. Investigation Phase (Analyst)

**Target**: $ARGUMENTS

1. **Map Symbols**: Use `scripts/symbol-index.sh` to understand the target's public interface.
2. **Scan Vulnerabilities**:
   - **Logic**: Edge cases, state consistency, missing error handling.
   - **Security**: Sanitization, authorization, secrets, resource leaks.
   - **Performance**: O(N^2) loops, lock contention, sync-in-async.
   - **Suspicious**: Hidden side effects, magic values.

## 2. Adversarial Audit (Critic)

**CRITICAL: Every finding must be verified by the `code-critic` agent.**

- The critic must attempt to find reasons why the code is "Correct as Intended".
- Only "Indisputable Bugs" or "Validated Risks" should remain.

## 3. Evidence & Test Generation

1. **Reproduction Plan**: For each finding, plan a minimal unit test.
2. **Generate Tests**: Create the test files (e.g., Go `_test.go` or Python `pytest`).
3. **Report**: Create `memory-bank/reports/audit_YYYYMMDD_topic.md`.

## Success Criteria
- [ ] Comprehensive audit report generated.
- [ ] Logic/Security/Performance findings verified by Critic.
- [ ] Unit tests generated for all CRITICAL/WARNING findings.
