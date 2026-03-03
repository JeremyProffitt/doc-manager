# Doc-Manager Build Prompt

Paste everything below the line after `/clear`.

---

Build the Doc-Manager application in `/c/dev/doc-manager/`. The repo already exists with a complete PRD (`prd.md`) and implementation plan (`plan.md`). Read both files fully before starting — they are your source of truth for all requirements, architecture, and build methodology.

## What you're building

An online document management tool where users upload blank forms (PDF/images), AI (Bedrock Sonnet) detects where standard fields go, users edit field placements in a drag-and-drop editor, then select a customer to auto-populate and download the filled form as a PDF. Tech stack: GoFiber + Lambda + DynamoDB + S3 + SAM + GitHub Actions.

## How to build it

**You are the orchestrator.** You coordinate a team of specialized subagents. Follow the plan exactly.

### Build methodology: Strict TDD

For every module — write tests FIRST, run them to confirm they fail, then write the implementation, then run tests to confirm they pass. No exceptions.

### Agent team

Launch these as parallel subagents (using the Agent tool with `isolation: "worktree"`) wherever the plan shows parallel work:

1. **Staff Developer** — Core Go code: models, store layer, handlers, services, templates, JS, PDF generation. Prompt this agent with the persona: "You are a staff-level Go developer. Write clean, idiomatic Go. Follow strict TDD — write tests first, then implementation. Use table-driven tests. Target 80%+ coverage."

2. **Staff Security Engineer** — Auth system, bcrypt, JWT sessions, input validation, CORS, pre-signed URL scoping, OWASP review. Prompt: "You are a staff-level application security engineer. Review and write code with a security-first mindset. Check for injection, auth bypass, session fixation, XSS, secret exposure. Write security-focused tests."

3. **Staff SRE** — DynamoDB table design, Lambda config, timeouts, error handling, cold start optimization, monitoring. Prompt: "You are a staff-level SRE. Focus on reliability, failure modes, resource limits, TTL correctness, retry behavior, and observability. Review infrastructure config for production readiness."

4. **Staff DevOps** — SAM template, GitHub Actions pipeline, seed script, Playwright CI wiring, deployment automation. Prompt: "You are a staff-level DevOps engineer. Build reliable CI/CD pipelines, IaC templates, and deployment automation. Ensure secrets are handled safely, artifacts managed correctly, and rollback is possible."

5. **Staff UI Engineer** — HTML/CSS mockups with Tailwind. Prompt: "You are a staff-level UI engineer. Build pixel-accurate, self-contained HTML mockups using Tailwind CSS via CDN. Include all states: empty, loading, error, populated. Use realistic mock data."

6. **Staff QA Engineer** — Reviews mockups and writes Playwright E2E tests. Prompt: "You are a staff-level QA engineer. Review UI mockups for usability issues, missing states, accessibility, and edge cases. Write comprehensive Playwright E2E tests covering happy paths, error paths, and edge cases."

### Execution order (follow the plan's dependency graph)

**Step 1: Scaffolding** — Launch DevOps, SRE, and Developer agents in parallel:
- DevOps: SAM template (`template.yaml`), GitHub Actions (`deploy.yml`), `samconfig.toml`, `Makefile`
- SRE: DynamoDB table schemas, S3 bucket config, IAM role policies
- Developer: `go mod init`, project directory structure, `config` package
- After merge: Security agent reviews IAM least-privilege and S3 bucket policy

**Step 1.5: UI Mockups [GATE]** — Launch UI Engineer and Developer agents in parallel:
- UI Engineer: Build all 9 HTML mockups in `mockups/` directory (login, dashboard, forms-list, forms-upload, forms-editor, forms-preview, customers-list, customers-edit, settings-fields)
- Developer: Review mockups for data/API alignment
- QA Engineer: Review all mockups for usability, missing states, edge cases
- **STOP and show me the mockups for approval before proceeding**

**Step 2: Authentication** — After mockup approval:
- Developer: Write auth tests first → implement auth handlers, session store, user store, middleware
- Security: Co-author bcrypt config, JWT security, cookie flags, input sanitization
- DevOps: Write seed script (`cmd/seed/main.go`) with TDD
- SRE: Sessions table TTL config, error logging

**Step 3: Core Backend** — After auth:
- Developer: Write store tests → implement DynamoDB stores (forms, customers, settings), route handlers, templates, S3 pre-signed upload flow
- Security: Review all handlers for injection, validate pre-signed URL scope
- SRE: Integration test infra (DynamoDB Local), Lambda timeout tuning
- DevOps: Add integration test step to CI pipeline

**Step 4: AI Integration** — After core backend:
- Developer: Write Bedrock service tests (mock client) → implement Bedrock service, analysis orchestration
- SRE: Timeout/retry tuning for Bedrock calls
- Security: Review for prompt injection risks

**Step 5: Form Editor** — After AI integration:
- Developer: Write editor handler tests → implement editor, version history, font config UI
- Security: Review editor for XSS (user field names in DOM)
- DevOps: Scaffold Playwright E2E tests, create test fixtures
- QA: Write `editor.spec.ts` and `versioning.spec.ts`

**Step 6: Population & Download** — After editor:
- Developer: Write PDF service tests → implement PDF generation with font support
- SRE: Profile PDF generation memory/timeout
- QA: Write `populate.spec.ts`

**Step 7: Full Test Suite** — After all features:
- Developer: Fill coverage gaps to 80%+
- Security: Final OWASP review pass on all routes
- DevOps: Full pipeline run: build → unit tests → integration tests → deploy → Playwright E2E
- Fix any failures, re-run until green

### Key technical details (from the PRD)

- **Auth:** bcrypt passwords, JWT session cookies, DynamoDB Sessions table with TTL. Seed user: `proffitt.jeremy@gmail.com` / `Docs4President!`
- **Upload:** Browser → pre-signed S3 PUT URL (5 min expiry) → S3 direct. No files through Lambda.
- **DynamoDB tables:** DocMgr-Users, DocMgr-Forms, DocMgr-FieldPlacements (PK: formId, SK: version), DocMgr-Customers, DocMgr-Settings, DocMgr-Sessions
- **Bedrock model:** `anthropic.claude-sonnet-4-20250514` in `us-east-1`
- **Font config:** Form-level defaults (family + size), field-level overrides (null = inherit)
- **Versioning:** Every field placement save creates a new immutable version. Revert copies old version as new version.
- **PDF:** Overlay text on original form at field coordinates using configured fonts
- **Lambda:** GoFiber with `fiberadaptor`, `provided.al2023`, 512MB, 300s timeout
- **GitHub secrets already configured:** AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY, JWT_SECRET, SEED_USER_PASSWORD
- **GitHub variables already configured:** AWS_REGION=us-east-1, SAM_S3_BUCKET=doc-manager-sam

### Merge protocol

After each agent completes work in its worktree:
1. Merge the worktree branch into main
2. Run `go test ./...` to verify nothing broke
3. If conflicts, resolve them
4. Move to the next step

Start now. Begin with Step 1 — read `prd.md` and `plan.md`, then launch the scaffolding agents in parallel.
