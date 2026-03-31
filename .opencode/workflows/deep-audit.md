---
description: Perform a deep code audit using multi-agent adversarial logic (Analysis + Review).
---

# Deep Audit Workflow

Use this workflow to diagnostic specific files or features with high precision. It bypasses the full RPEQ cycle and focuses on **Code Quality and Logic Consistency**.

## Phase 1: Context Research (Analyst)

1. **Map Entry Points**: Identify public APIs, main functions, and configuration points.
2. **Deep Logic Trace**: Use the `codebase-analyzer` agent or manual tracing to understand the intended data flow.
3. **Draft Preliminary Findings**: List all potential bugs, security risks, and optimization points.

## Phase 2: Adversarial Audit (Critic)

**CRITICAL: Every preliminary finding must be audited by the `code-critic` subagent.**

1. **Task**: "Review these findings and try to disprove them. Search for mitigating context in the codebase."
2. **Execution**: The critic must attempt to find reasons why the code is safe or intentional.
3. **Consensus**: Only findings that the critic **VERIFIES** are allowed to proceed to the final report.

## Phase 3: Final Evidence Report

1. **Categorize**: Group findings by severity (CRITICAL, WARNING, INFO).
2. **Reproduction Path**: For every CRITICAL bug, provide a step-by-step logic path to trigger the error.
3. **Evidence**: Include file:line references for both the bug and any mitigating factors found.

## Summary

This workflow ensures that you don't waste time on false alarms. It forces the AI to "think twice" and "act as its own skeptic" before presenting results to the user.
