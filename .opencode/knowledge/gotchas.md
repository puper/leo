# Project Gotchas & Knowledge Base

This file contains **Hard Lessons** and **Codebase-Specific Patterns** discovered during AI-driven development. It exists so the Agent never makes the same mistake twice.

---

## 🏗️ Architecture & Conventions

### [Example Gotcha: Auth Middleware]
- **Scenario**: Adding a new API endpoint.
- **The Issue**: All `/api/*` routes MUST be registered in `src/app.ts` *before* the `authMiddleware` or they bypass security.
- **The Lesson**: Always place new routes after line 45 in `src/app.ts`.

---

## 🗄️ Database & State

### [Example Gotcha: Transaction Scope]
- **Scenario**: Multi-step user creation.
- **The Issue**: The `UserStore` does not automatically roll back if the `ProfileStore` fails.
- **The Lesson**: Use the `UnitOfWork` pattern in `services/user-service.ts` for any multi-store operation.

---

## 🛠️ Testing & Harness

### [Example Gotcha: Deterministic Snapshots]
- **Scenario**: Updating API response snapshots.
- **The Issue**: Timestamps and UUIDs in responses cause test failures.
- **The Lesson**: Use the `normalize(obj)` helper in `tests/utils.py` to redact dynamic fields before asserting snapshots.

---

## 📝 How to Update

1.  **Context**: What was the task?
2.  **Problem**: What failed or caused a lot of back-and-forth?
3.  **Solution**: What is the definitive way to do it correctly next time?
4.  **Reference**: Link to the target file or execution log.
