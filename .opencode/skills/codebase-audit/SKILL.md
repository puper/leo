---
name: codebase-audit
description: Perform a comprehensive audit of existing code for logic bugs, suspicious patterns, serious security vulnerabilities, and performance bottlenecks. Includes automatic unit test generation for findings.
compatibility: opencode
license: MIT
metadata:
  writes-to: memory-bank/reports/
  allowed-tools: Read, Grep, Glob, Bash, Agent
  hard-guards: Fact-based logic only; No fixing in audit phase; Audit all paths (Logic, Security, Performance)
---

# Codebase Audit

Analyze existing codebase with high precision. This skill detects bugs, security risks, and performance bottlenecks, then plans unit tests to prove them.

## 1. Audit Checklist

### 1.1 Logic & Functional Bugs
| Category | Check |
|----------|-------|
| Boundary | Are min/max values and edge cases handled? |
| Null/Empty | Are null pointers or empty inputs pre-validated? |
| State | Is the system state consistent across multi-step operations? |
| Logic Paths | Are there unreachable branches or missing error returns? |

### 1.2 Serious Security Issues
| Category | Check |
|----------|-------|
| Sanitization | Is user input sanitized to prevent injection (SQL, XSS, etc.)? |
| Auth/Authz | Are authentication and authorization checks bypassed? |
| Secrets | Are there hardcoded secrets or sensitive data leaks? |
| Resource | Are files or connections leaking? |

### 1.3 Performance Bottlenecks
| Category | Check |
|----------|-------|
| Efficiency | Are there O(N^2) loops or redundant allocations? |
| Concurrency | Are there potential deadlocks or excessive lock contention? |
| Latency | Any synchronous blocking calls in async paths? |

### 1.4 Suspicious Code
| Category | Check |
|----------|-------|
| Magic Values | Undocumeted constants or hardcoded IDs? |
| Hidden Side Effects | Does a getter modify state? |
| Complexity | Is a function too complex to be verified? |

## 2. Audit Workflow

### Step 1: Mapping & Trace
- Run `scripts/symbol-index.sh` for context.
- Identify the target file/function's entry and exit points.
- Map internal data flow and external dependencies.

### Step 2: Adversarial Review (Council Logic)
- **Identify** a potential issue.
- **Audit**: Use a `code-critic` agent to attempt to disprove the issue.
- **Verify**: Only report if the issue is logically sound and reproduction is possible.

### Step 3: Test Generation Plan
- For every CRITICAL or WARNING finding:
- Plan a **Unit Test** that triggers the failure.
- Specify the test file and the input/assertion required.

### Step 4: Write Audit Report
Create `memory-bank/reports/audit_YYYYMMDD_HHMMSS_<topic>.md`:

```markdown
---
topic: "<topic>"
verdicts: [Logic, Security, Performance]
---
## Findings
### [Finding Name] (Severity: CRITICAL)
- **Description**: [Summary]
- **Impact**: [Security/Logic/Perf impact]
- **Proof**: [Evidence path]
- **Reproduction**: [Step-by-step]

## Unit Test Evidence
### [Test Name]
- **Target File**: `path/to/test_file.ext`
- **Logic**: [How the test proves the bug]
```

## 3. Post-Audit: Test Execution
After generating the report, the agent **MUST** proceed to generate the actual test files listed in the report.
