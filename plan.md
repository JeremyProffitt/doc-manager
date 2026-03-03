# Doc-Manager: Implementation Plan

## Build Methodology: TDD with Agent Team

### Test-Driven Development (TDD) — Tests Before Code

Every implementation step follows strict TDD. **No production code is written until its tests exist and fail.**

For each feature/module:
1. **Red:** Write the test(s) first — unit tests for the module, handler tests for the endpoint, Playwright spec for the UI flow. Run them — they must fail (compilation errors count as "red" for Go).
2. **Green:** Write the minimum production code to make the tests pass.
3. **Refactor:** Clean up the code while keeping tests green.

This applies to every step below. When a step says "implement X", the actual sequence is:
1. Write tests for X
2. Run tests → confirm they fail
3. Implement X
4. Run tests → confirm they pass
5. Refactor if needed

### Agent Team — Parallel Subagent Execution

Building is done by a team of **4 specialized agent personas**, each running as a subagent. Work is parallelized across agents wherever steps have no dependencies. The orchestrator (main conversation) coordinates handoffs and merges.

#### Agent Personas

| Agent | Persona | Responsibilities | Review Focus |
|-------|---------|-----------------|--------------|
| **Staff Developer** | Senior application engineer | Core Go code: handlers, services, models, store layer, templates, JS editor, PDF generation | Code quality, patterns, test coverage, Go idioms |
| **Staff Security Engineer** | Application security specialist | Auth system, password hashing, session management, input validation, CORS, pre-signed URL security, OWASP review of all handlers | Injection prevention, auth bypass, session fixation, secret exposure, header security |
| **Staff SRE** | Reliability & observability engineer | Lambda configuration, DynamoDB table design, S3 lifecycle, CloudWatch alarms, error handling, cold start optimization, timeout tuning | Failure modes, retry behavior, TTL correctness, resource limits, monitoring gaps |
| **Staff DevOps** | CI/CD & infrastructure engineer | SAM template, GitHub Actions pipelines, seed script, Playwright CI setup, deployment automation, environment config | IaC correctness, pipeline reliability, secret handling, artifact management, deployment rollback |

#### Agent Coordination Model

```
Orchestrator (main conversation)
  │
  ├──> [Parallel] Step 1: Infrastructure
  │     ├── Staff DevOps:     SAM template, GH Actions pipeline, samconfig
  │     ├── Staff SRE:        DynamoDB table schemas, S3 bucket config, IAM policies
  │     ├── Staff Developer:  Go module init, project structure, config package
  │     └── Staff Security:   Review IAM least-privilege, S3 bucket policy, CORS
  │
  ├──> [Parallel] Step 2: Auth + Step 3 foundations (after Step 1 merge)
  │     ├── Staff Developer:  Auth tests → auth handlers, session store, middleware
  │     ├── Staff Security:   Auth tests → bcrypt config, JWT security, cookie flags,
  │     │                     input sanitization middleware, OWASP review
  │     ├── Staff DevOps:     Seed script (tests → implementation)
  │     └── Staff SRE:        Sessions table TTL, DynamoDB capacity mode, error logging
  │
  ├──> [Parallel] Step 3 continued + Step 4 (after auth merge)
  │     ├── Staff Developer:  Store layer tests → DynamoDB stores, route handlers,
  │     │                     templates, S3 pre-signed upload flow
  │     ├── Staff Security:   Review all handlers for injection, validate pre-signed URL
  │     │                     scope (PUT only, correct bucket/prefix, expiry)
  │     ├── Staff SRE:        Integration test infrastructure (DynamoDB Local in Docker),
  │     │                     Lambda timeout tuning for Bedrock calls
  │     └── Staff DevOps:     Integration test pipeline step, DynamoDB Local in CI
  │
  ├──> [Parallel] Step 5 + Step 6 (after core backend merge)
  │     ├── Staff Developer:  Editor JS tests → editor implementation, version history,
  │     │                     font config UI, PDF service tests → PDF generation
  │     ├── Staff Security:   Review editor for XSS (user-controlled field names rendered
  │     │                     in DOM), validate version revert can't access other users' data
  │     ├── Staff SRE:        PDF generation memory/timeout profiling, S3 download URL
  │     │                     expiry validation, version accumulation monitoring
  │     └── Staff DevOps:     Playwright E2E scaffold, test fixture files, CI E2E job
  │
  ├──> [Sequential] Step 7: Full test suite execution
  │     ├── Staff Developer:  Fill coverage gaps, fix failing tests
  │     ├── Staff Security:   Final security review pass — all routes, all inputs
  │     ├── Staff SRE:        Verify monitoring, error paths, timeout behavior
  │     └── Staff DevOps:     Full pipeline run: build → test → deploy → E2E
  │
  └──> Final: All agents review → merge → deploy → E2E green
```

#### UI Mockup Team — Human-in-the-Loop Gate

Before any frontend implementation begins (Steps 3-6), a **UI Mockup Team** builds interactive HTML mockups of every page. These mockups are reviewed and approved by the user before production code is written.

| Agent | Persona | Responsibilities |
|-------|---------|-----------------|
| **Staff UI Engineer** | Frontend design & UX specialist | Builds pixel-accurate HTML/CSS mockups using Tailwind, ensures consistent design system, navigation flow, responsive layout |
| **Staff Developer** | Application engineer (shared with build team) | Ensures mockups use realistic data structures, validates that the UI maps to actual API routes and form field models |
| **Staff QA Engineer** | Quality assurance & usability reviewer | Reviews each mockup for usability issues, missing states (empty, loading, error), accessibility basics, and edge cases (long text, many items) |

**Mockup process:**
1. UI Engineer + Developer build HTML mockups in parallel (one page per agent call where possible)
2. QA Engineer reviews all mockups and files issues
3. All mockups are placed in `mockups/` directory as standalone HTML files (self-contained, viewable in a browser)
4. **Orchestrator presents mockups to the user for approval** — this is a hard gate; no frontend code proceeds until approved
5. Approved mockups become the reference spec for the Developer agent during implementation

**Pages requiring mockups:**

| Page | File | Key Elements to Validate |
|------|------|-------------------------|
| Login | `mockups/login.html` | Email/password fields, error states, branding |
| Dashboard | `mockups/dashboard.html` | Nav bar, quick actions, recent forms grid, empty state |
| Form Library | `mockups/forms-list.html` | Form cards with thumbnails, status badges, search/filter, empty state |
| Form Upload | `mockups/forms-upload.html` | Drag-and-drop zone, progress bar, file type validation |
| Form Editor | `mockups/forms-editor.html` | Canvas with field overlays, fields panel, font controls, version history sidebar, field properties panel |
| Form Populate Preview | `mockups/forms-preview.html` | Form with customer data overlaid, customer selector, download button |
| Customer List | `mockups/customers-list.html` | Customer table/cards, add/edit/delete actions, empty state |
| Customer Add/Edit | `mockups/customers-edit.html` | Form with all standard fields, validation errors |
| Settings — Fields | `mockups/settings-fields.html` | Field list, add/remove/rename, drag to reorder |

**Mockup coordination in the build flow:**
```
Step 1: Scaffolding (no UI)
  │
  ├──> Step 1.5: UI Mockups ← NEW GATE
  │     ├── [Parallel] Staff UI Engineer: Build all 9 mockups
  │     ├── [Parallel] Staff Developer: Review data/API alignment
  │     ├── [Sequential] Staff QA Engineer: Review all mockups
  │     └── [GATE] User approves mockups before proceeding
  │
  ├──> Step 2: Auth (backend only, login page follows approved mockup)
  ...
```

#### Subagent Usage Guidelines

- **Parallel launches:** When two or more agents have independent tasks, launch them simultaneously using multiple Agent tool calls in a single message
- **Worktree isolation:** Each agent works in a git worktree (`isolation: "worktree"`) to avoid conflicts
- **Review handoffs:** After an agent completes work, a different agent reviews it (developer writes code → security reviews it; DevOps writes pipeline → SRE reviews it)
- **Merge protocol:** Orchestrator merges worktree branches sequentially, resolving conflicts, running full test suite after each merge
- **Escalation:** If an agent is blocked or discovers a cross-cutting concern, it returns to the orchestrator who coordinates with the relevant agent

---

## Phase 1 Implementation Roadmap

### Step 1: Project Scaffolding & Infrastructure

**Goal:** Set up the Go project structure, SAM template (with DynamoDB tables), and GitHub Actions pipeline.
**Agents:** DevOps (lead), SRE, Developer, Security (review)
**TDD:** Write `sam validate` check and a basic `go build` CI step first — pipeline must fail initially, then pass after scaffolding is complete.

#### 1.1 Go Project Setup
- Initialize Go module (`go mod init github.com/JeremyProffitt/doc-manager`)
- Add dependencies:
  - `github.com/gofiber/fiber/v2` — web framework
  - `github.com/gofiber/adaptor/v2` — Lambda adapter
  - `github.com/aws/aws-sdk-go-v2` — AWS SDK (Bedrock, S3, DynamoDB)
  - `github.com/aws/aws-lambda-go` — Lambda runtime
  - `github.com/jung-kurt/gofpdf` or `github.com/unidoc/unipdf` — PDF generation
  - `golang.org/x/crypto/bcrypt` — password hashing
  - `github.com/golang-jwt/jwt/v5` — session tokens

#### 1.2 Project Directory Structure
```
doc-manager/
├── .github/
│   └── workflows/
│       └── deploy.yml              # GitHub Actions CI/CD pipeline
├── cmd/
│   ├── lambda/
│   │   └── main.go                 # Lambda entry point
│   └── seed/
│       └── main.go                 # Seed script (create user, mock customers)
├── internal/
│   ├── handlers/
│   │   ├── auth.go                 # Login, logout, session middleware
│   │   ├── forms.go                # Form upload, list, view, delete handlers
│   │   ├── editor.go               # Form editor page handlers
│   │   ├── populate.go             # Form population & download handlers
│   │   ├── customers.go            # Customer CRUD handlers
│   │   ├── settings.go             # Standard fields config handlers
│   │   ├── versions.go             # Field placement version handlers
│   │   └── home.go                 # Dashboard handler
│   ├── middleware/
│   │   └── auth.go                 # Auth middleware (session validation)
│   ├── models/
│   │   ├── form.go                 # Form data structures (incl. font settings)
│   │   ├── customer.go             # Customer data structures
│   │   ├── field.go                # Field placement + versioning structures
│   │   ├── user.go                 # User data structures
│   │   └── session.go              # Session data structures
│   ├── services/
│   │   ├── bedrock.go              # Bedrock AI integration
│   │   ├── s3.go                   # S3 file storage + pre-signed URL generation
│   │   ├── pdf.go                  # PDF rendering with font config support
│   │   └── analysis.go             # Form analysis orchestration
│   ├── store/
│   │   ├── dynamo.go               # DynamoDB client initialization
│   │   ├── user_store.go           # User CRUD (DynamoDB Users table)
│   │   ├── form_store.go           # Form CRUD (DynamoDB Forms table)
│   │   ├── field_store.go          # Field placements + versioning (DynamoDB)
│   │   ├── customer_store.go       # Customer CRUD (DynamoDB Customers table)
│   │   ├── session_store.go        # Session management (DynamoDB Sessions table)
│   │   └── settings_store.go       # Settings CRUD (DynamoDB Settings table)
│   └── config/
│       └── config.go               # Application configuration
├── templates/
│   ├── layouts/
│   │   └── base.html               # Base layout (head, nav, footer)
│   ├── login.html                  # Login page
│   ├── home.html                   # Dashboard page
│   ├── forms/
│   │   ├── list.html               # Form library
│   │   ├── view.html               # Single form view
│   │   └── editor.html             # Form editor (canvas + fields + versions)
│   ├── customers/
│   │   ├── list.html               # Customer list
│   │   ├── view.html               # Customer detail
│   │   └── edit.html               # Customer add/edit
│   ├── populate/
│   │   └── preview.html            # Populated form preview
│   └── settings/
│       └── fields.html             # Standard fields configuration
├── static/
│   ├── css/
│   │   └── app.css                 # Custom styles
│   └── js/
│       ├── editor.js               # Form editor canvas + version history
│       ├── upload.js               # Pre-signed URL upload flow
│       └── app.js                  # General UI interactions
├── mockups/                        # HTML mockups (user-approved before implementation)
│   ├── login.html
│   ├── dashboard.html
│   ├── forms-list.html
│   ├── forms-upload.html
│   ├── forms-editor.html
│   ├── forms-preview.html
│   ├── customers-list.html
│   ├── customers-edit.html
│   └── settings-fields.html
├── e2e/                            # Playwright end-to-end tests
│   ├── playwright.config.ts        # Playwright config (baseURL from env)
│   ├── package.json                # Node deps (playwright, @playwright/test)
│   ├── tsconfig.json
│   ├── fixtures/
│   │   ├── sample-form.pdf         # Test PDF for upload flows
│   │   └── sample-form.png         # Test image for upload flows
│   ├── helpers/
│   │   └── auth.ts                 # Shared login helper / storage state
│   └── tests/
│       ├── auth.spec.ts            # Login/logout/session tests
│       ├── forms.spec.ts           # Form upload, library, delete tests
│       ├── editor.spec.ts          # Field editor drag/drop/resize/font tests
│       ├── versioning.spec.ts      # Version create/list/revert tests
│       ├── customers.spec.ts       # Customer CRUD tests
│       ├── populate.spec.ts        # Form population & PDF download tests
│       ├── settings.spec.ts        # Standard fields config tests
│       └── smoke.spec.ts           # Quick smoke test (login → dashboard loads)
├── template.yaml                   # AWS SAM template
├── samconfig.toml                  # SAM deployment config
├── Makefile                        # Build and deploy commands
├── prd.md
├── plan.md
├── go.mod
└── go.sum
```

#### 1.3 AWS SAM Template (`template.yaml`)

Resources to define:

**Lambda Function**
- Runtime: `provided.al2023`
- Handler: `bootstrap`
- Memory: 512MB
- Timeout: 300s (accommodates AI analysis)
- Architecture: `x86_64`
- Environment variables: S3 bucket name, Bedrock model ID, AWS region, JWT secret, DynamoDB table names

**API Gateway v2** (HTTP API)
- `$default` route proxying all requests to Lambda

**S3 Bucket**
- For form file storage
- CORS configured for browser pre-signed URL uploads (PUT from the app's domain)

**DynamoDB Tables**

| Table | PK | SK | Notes |
|-------|----|----|-------|
| `DocMgr-Users` | `email` (S) | — | |
| `DocMgr-Forms` | `id` (S) | — | GSI on `userId` for per-user queries |
| `DocMgr-FieldPlacements` | `formId` (S) | `version` (N) | Sorted by version descending |
| `DocMgr-Customers` | `id` (S) | — | |
| `DocMgr-Settings` | `key` (S) | — | |
| `DocMgr-Sessions` | `token` (S) | — | TTL attribute for auto-expiry |

**IAM Role**
- Permissions: S3 (read/write bucket), DynamoDB (all tables), Bedrock (InvokeModel), CloudWatch Logs

#### 1.4 GitHub Actions Pipeline (`.github/workflows/deploy.yml`)
- **Trigger:** Push to `main` branch
- **Steps:**
  1. Checkout code
  2. Set up Go toolchain
  3. Run `go test ./...`
  4. Run `go vet ./...`
  5. Build binary: `GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o bootstrap cmd/lambda/main.go`
  6. Set up AWS credentials (GitHub secrets)
  7. Install SAM CLI
  8. Run `sam build`
  9. Run `sam deploy --no-confirm-changeset --no-fail-on-empty-changeset`
  10. Run seed script (only on first deploy or when `--seed` flag passed)
  11. Output the API Gateway URL

#### 1.5 Seed Script (`cmd/seed/main.go`)
- Connects to DynamoDB
- Creates the seed user (`proffitt.jeremy@gmail.com` / bcrypt hash of `Docs4President!`)
- Inserts the 5 mock customers from the PRD
- Inserts default standard field definitions into the Settings table
- Idempotent — uses `PutItem` with condition expressions to avoid overwriting existing data

---

### Step 2: Authentication Layer

**Goal:** Implement login/logout with DynamoDB-backed sessions.
**Agents:** Developer (implementation), Security (co-author + review), DevOps (seed script), SRE (session TTL)
**TDD sequence:**
1. Write `auth_test.go` — all handler tests (valid login, bad password, missing fields, logout)
2. Write `middleware/auth_test.go` — all middleware tests (valid cookie, no cookie, expired token)
3. Write `session_store_test.go` — store tests (create, validate, delete, expiry)
4. Write `user_store_test.go` — store tests (get user, not found)
5. Run all → all fail
6. Implement auth handlers, middleware, stores
7. Run all → all pass

#### 2.1 User Model (`internal/models/user.go`)
```go
type User struct {
    Email        string `dynamodbav:"email"`
    PasswordHash string `dynamodbav:"passwordHash"`
    Name         string `dynamodbav:"name"`
    CreatedAt    string `dynamodbav:"createdAt"`
}
```

#### 2.2 Auth Handlers (`internal/handlers/auth.go`)
- `GET /login` — render login page
- `POST /login` — validate email/password against DynamoDB, bcrypt compare; on success, create a session token (JWT signed with a secret), store in DynamoDB Sessions table with TTL, set as HTTP-only secure cookie
- `POST /logout` — delete session from DynamoDB, clear cookie

#### 2.3 Auth Middleware (`internal/middleware/auth.go`)
- Runs on all routes except `/login`, `/static/*`
- Reads session token from cookie
- Validates token signature and checks existence in DynamoDB Sessions table
- If invalid/expired, redirects to `/login`
- If valid, attaches user info to Fiber context (`c.Locals("user", user)`)

---

### Step 3: Backend — Core Application

**Goal:** Build the GoFiber application with routing, templates, and DynamoDB data layer.
**Agents:** Developer (lead), Security (input validation review), SRE (integration test infra), DevOps (CI integration tests)
**TDD sequence:**
1. Write `form_store_test.go` — CRUD tests for forms
2. Write `customer_store_test.go` — CRUD tests for customers
3. Write `settings_store_test.go` — settings get/update tests
4. Write `forms_test.go` — handler tests for upload URL, complete, list, view, delete
5. Write `customers_test.go` — handler tests for full CRUD
6. Write `s3_test.go` — pre-signed URL generation tests
7. Run all → all fail
8. Implement stores, handlers, S3 service, templates
9. Run all → all pass

#### 3.1 Lambda Entry Point (`cmd/lambda/main.go`)
- Initialize Fiber app with routes
- Initialize DynamoDB client, S3 client
- Register auth middleware
- Detect environment (Lambda vs local) for development flexibility
- Use `fiberadaptor.New(app)` for Lambda, `app.Listen(":3000")` for local dev

#### 3.2 DynamoDB Store Layer (`internal/store/`)
- Each store file implements CRUD for its table
- `form_store.go`:
  - `CreateForm(form)` — PutItem
  - `GetForm(id)` — GetItem
  - `ListForms(userId)` — Query on GSI
  - `UpdateForm(form)` — UpdateItem
  - `DeleteForm(id)` — DeleteItem + delete S3 file + delete all FieldPlacements versions
- `field_store.go`:
  - `SaveFieldPlacement(formId, placement)` — auto-increments version, PutItem
  - `GetLatestFieldPlacement(formId)` — Query with ScanIndexForward=false, Limit=1
  - `GetFieldPlacement(formId, version)` — GetItem
  - `ListVersions(formId)` — Query all versions, return metadata only (no field data)
  - `RevertToVersion(formId, version)` — reads old version, saves as new version with source `revert_from_vN`
- `customer_store.go`:
  - Full CRUD for customers

#### 3.3 S3 Service (`internal/services/s3.go`)
- `GenerateUploadURL(formId, contentType)` — returns pre-signed PUT URL (5 min expiry)
- `GenerateDownloadURL(s3Key)` — returns pre-signed GET URL (15 min expiry)
- `GetObject(s3Key)` — fetch file bytes (for Bedrock analysis and PDF generation)
- `DeleteObject(s3Key)` — delete form file
- S3 bucket CORS: allow PUT from `*` origin (tighten to app domain in production)

#### 3.4 Route Handlers (`internal/handlers/`)
- Implement all routes from the PRD API table
- Each page handler renders the appropriate Go HTML template
- API routes return JSON responses
- Upload flow:
  1. `POST /api/forms/upload-url` — creates form record in DynamoDB, returns pre-signed URL + form ID
  2. Browser uploads directly to S3
  3. `POST /api/forms/:id/upload-complete` — updates form status to `uploaded`

#### 3.5 HTML Templates (`templates/`)
- Base layout with navigation (Forms, Customers, Logout links)
- Login page (no nav bar)
- Tailwind CSS via CDN for styling
- Server-side rendered pages (minimal JS except for the editor and upload)

---

### Step 4: AI Integration — Bedrock Form Analysis

**Goal:** Integrate with Amazon Bedrock to analyze uploaded forms.
**Agents:** Developer (lead), SRE (timeout/retry tuning), Security (prompt injection review)
**TDD sequence:**
1. Write `bedrock_test.go` — mock client tests (valid response, malformed JSON, empty response, low confidence)
2. Write `analysis_test.go` — orchestration tests (single page, multi-page, partial failure)
3. Write `field_store_test.go` — version 1 creation from AI results
4. Run all → all fail
5. Implement Bedrock service, analysis orchestration, field store save
6. Run all → all pass

#### 4.1 Bedrock Service (`internal/services/bedrock.go`)
- Use AWS SDK v2 Bedrock Runtime client
- Model: `anthropic.claude-sonnet-4-20250514` (or latest Sonnet available on Bedrock)
- Fetch form from S3, convert to base64
- Construct structured prompt:
  ```
  Analyze this form image. Identify the locations where the following fields
  should be filled in: [Name, Business, Address, City, State, Zip, Phone Number].

  For each field, return a JSON array with:
  - "field_name": the name of the field
  - "page": page number (1-indexed)
  - "x": horizontal position as a percentage (0-100) from the left edge
  - "y": vertical position as a percentage (0-100) from the top edge
  - "width": field width as a percentage of form width
  - "height": field height as a percentage of form height
  - "confidence": confidence score 0.0 to 1.0
  - "reasoning": brief explanation of why this location was chosen

  Return ONLY valid JSON. Do not include any other text.
  ```
- Parse the response JSON
- Store as version 1 in `FieldPlacements` table with source `ai_analysis`

#### 4.2 Analysis Orchestration (`internal/services/analysis.go`)
- Handle PDF-to-image conversion (if needed) using a Go library
- For multi-page PDFs, analyze each page separately
- Merge results into a single field placement set
- Handle errors gracefully (return partial results if some pages fail)
- This is a POC — accept that accuracy will vary; the editor handles corrections

---

### Step 5: Form Editor — Interactive Field Placement with Versioning

**Goal:** Build the interactive form editor with drag/drop, font config, and version history.
**Agents:** Developer (lead), Security (XSS review on editor DOM), SRE (version accumulation), DevOps (Playwright scaffold)
**TDD sequence:**
1. Write `field_store_test.go` additions — version increment, list versions, get specific version, revert
2. Write `editor_test.go` — handler tests for editor page, save fields, version API
3. Write `versions_test.go` — handler tests for version list, get, revert
4. Write `e2e/tests/editor.spec.ts` — Playwright tests for drag/resize/add/remove/font
5. Write `e2e/tests/versioning.spec.ts` — Playwright tests for version list and revert
6. Run Go tests → all fail; Playwright tests saved for post-deploy
7. Implement editor handlers, version handlers, JS editor, templates
8. Run Go tests → all pass

#### 5.1 Editor Frontend (`static/js/editor.js`)
- Load the form image as a canvas background (via pre-signed S3 GET URL)
- Fetch current field placements from `/api/forms/:id/fields`
- Render draggable, resizable field overlays using HTML `div` elements
- Implement:
  - **Drag:** Mouse/touch drag to reposition fields
  - **Resize:** Corner handles to resize field bounding boxes
  - **Add:** Click "Add Field" button, select field type, click on form to place
  - **Remove:** Click field, press delete or click remove button
  - **Font Settings:** Form-level font family and size pickers in the toolbar; per-field override inputs in the field properties panel (shown when a field is selected)
  - **Save:** Serialize all field positions + font settings and PUT to `/api/forms/:id/fields` — this creates a new version
- Color-code fields by type for easy identification
- Percentage-based coordinates ensure responsiveness

#### 5.2 Version History Panel
- Right sidebar or dropdown showing all versions
- Each entry shows: version number, timestamp, source (AI / manual edit / revert)
- Click a version to preview that layout (read-only overlay)
- "Revert to this version" button creates a new version with the old layout
- Fetch versions from `/api/forms/:id/fields/versions`

#### 5.3 Editor Page (`templates/forms/editor.html`)
- Left sidebar: list of fields (checkboxes to show/hide)
- Main area: form image with field overlays
- Top bar: form name, page navigation, form-level font settings, Save/Cancel buttons
- Right sidebar: version history
- Bottom/side panel: selected field properties (position, size, font overrides)

---

### Step 6: Form Population & Download

**Goal:** Populate forms with customer data and generate downloadable PDFs with correct fonts.
**Agents:** Developer (lead), SRE (PDF memory/timeout profiling), Security (file output validation)
**TDD sequence:**
1. Write `pdf_test.go` — PDF generation tests (image form, PDF form, font defaults, field overrides, empty fields, page count)
2. Write `populate_test.go` — handler tests (preview, download, 404 cases, no-fields case)
3. Write `e2e/tests/populate.spec.ts` — Playwright tests for preview and download
4. Run Go tests → all fail
5. Implement PDF service, populate handlers, preview template
6. Run Go tests → all pass

#### 6.1 PDF Service (`internal/services/pdf.go`)
- Load the original form from S3 (PDF or image)
- Get the latest field placements from DynamoDB
- For each field:
  - Determine effective font: field-level override if set, otherwise form-level default
  - Render the customer's data value at the specified coordinates using the determined font family and size
- For PDF forms: overlay text on existing PDF pages
- For image forms: render text on the image, then convert to PDF
- Return the generated PDF as a byte stream

#### 6.2 Population Flow
1. User selects a form and a customer
2. Preview page shows the form with customer data rendered in the browser (JS overlays text on the form image using font settings)
3. User clicks "Download" to get the PDF
4. Backend generates PDF with customer data at stored field positions with correct fonts

---

### Step 7: Testing

Testing is split into three tiers: unit tests (Go), integration tests (Go, against local/mock AWS), and end-to-end tests (Playwright, against the deployed production URL). Target: **80%+ code coverage** for Go unit + integration tests.

---

#### 7.1 Unit Tests (Go)

Run with `go test ./...`. Each package gets its own `_test.go` files. Use interfaces and dependency injection so AWS services can be mocked without hitting real infrastructure.

**Store Layer** (`internal/store/*_test.go`)
- `user_store_test.go`
  - Create user, get by email, get non-existent user returns nil
  - Duplicate email returns error (conditional put)
  - Password hash is stored, not plaintext
- `form_store_test.go`
  - Create form, get by ID, list by user ID
  - Update form metadata (name, status, font settings)
  - Delete form removes record
  - List returns empty slice (not nil) when no forms exist
- `field_store_test.go`
  - Save placement auto-increments version
  - GetLatest returns highest version number
  - GetVersion returns exact version
  - ListVersions returns metadata sorted descending
  - Revert copies old version data into new version with correct source tag
  - Revert from non-existent version returns error
  - Concurrent saves produce sequential versions (no gaps/duplicates)
- `customer_store_test.go`
  - Full CRUD cycle
  - Delete non-existent customer returns error
  - List returns all seeded customers
- `session_store_test.go`
  - Create session, validate token, delete session
  - Expired session (past TTL) returns invalid
  - Non-existent token returns invalid
- `settings_store_test.go`
  - Get default fields, update fields, add custom field, remove field

**Mock strategy:** Use a DynamoDB interface wrapper so tests can use an in-memory fake or `dockertest` with DynamoDB Local.

**Services** (`internal/services/*_test.go`)
- `bedrock_test.go`
  - Mock Bedrock client returns valid JSON → parsed correctly into `[]FieldPlacement`
  - Bedrock returns malformed JSON → error handled, no panic
  - Bedrock returns empty response → returns empty field list
  - Bedrock returns fields with low confidence → all fields preserved, confidence scores intact
  - Prompt includes all configured standard fields
- `s3_test.go`
  - GenerateUploadURL returns valid pre-signed URL with correct expiry
  - GenerateDownloadURL returns valid pre-signed URL
  - GetObject returns file bytes
  - DeleteObject succeeds / handles missing key gracefully
- `pdf_test.go`
  - Generates PDF from image-based form with text overlays
  - Generates PDF from PDF-based form with text overlays
  - Form-level font settings applied to all fields
  - Field-level font override applied only to that field
  - Field with null font override inherits form defaults
  - Empty customer field value → no text rendered at that position
  - Output PDF has correct number of pages
- `analysis_test.go`
  - Single-page image analyzed correctly
  - Multi-page PDF: each page analyzed, results merged
  - Partial failure (one page fails) returns results for successful pages
  - Unsupported file type returns error

**Handlers** (`internal/handlers/*_test.go`)

Use `httptest` with the Fiber app to test HTTP request/response cycles. Mock the store and service layers.

- `auth_test.go`
  - POST /login with valid credentials → 302 redirect to `/`, session cookie set
  - POST /login with wrong password → 401, error message rendered
  - POST /login with non-existent email → 401, same error message (no user enumeration)
  - POST /login with empty fields → 400 validation error
  - POST /logout → session deleted, cookie cleared, redirect to /login
  - GET /login when already authenticated → redirect to /
- `forms_test.go`
  - POST /api/forms/upload-url → returns JSON with presigned URL and form ID
  - POST /api/forms/:id/upload-complete → updates form status to "uploaded"
  - GET /forms → renders form library page with all user forms
  - GET /forms/:id → renders form detail page
  - DELETE /forms/:id → deletes form, S3 object, and all versions
  - DELETE /forms/:id for non-existent form → 404
- `editor_test.go`
  - GET /forms/:id/edit → renders editor page with current field data
  - PUT /api/forms/:id/fields with valid payload → creates new version, returns 200
  - PUT /api/forms/:id/fields with empty fields array → accepted (clears fields)
  - PUT /api/forms/:id/fields with invalid JSON → 400
- `versions_test.go`
  - GET /api/forms/:id/fields/versions → returns version list
  - GET /api/forms/:id/fields/:version → returns specific version data
  - POST /api/forms/:id/fields/revert/:v → creates new version from old, returns new version number
  - Revert to non-existent version → 404
- `customers_test.go`
  - Full CRUD: create → read → update → delete
  - Create with missing required fields → 400
  - GET /api/customers → returns JSON array
  - DELETE non-existent customer → 404
- `populate_test.go`
  - GET /forms/:id/populate/:custId → renders preview page
  - GET /forms/:id/download/:custId → returns PDF with correct Content-Type header
  - Populate with non-existent form → 404
  - Populate with non-existent customer → 404
  - Populate form with no field placements → returns original form (no overlays)
- `settings_test.go`
  - GET /settings/fields → renders settings page with current fields
  - PUT /api/settings/fields → updates field definitions

**Middleware** (`internal/middleware/auth_test.go`)
- Request with valid session cookie → passes through, user in context
- Request with no cookie → 302 redirect to /login
- Request with expired/invalid token → 302 redirect to /login
- Request to /login (exempt path) → passes through without auth check
- Request to /static/* (exempt path) → passes through without auth check

---

#### 7.2 Integration Tests (Go)

Run with `go test -tags=integration ./...` (build tag gated so they don't run in normal `go test`). These test the full stack within Go — real HTTP requests through the Fiber app with real (local) DynamoDB.

**Setup:** Use `docker-compose` with DynamoDB Local for CI, or the `dynamodb-local` JAR. A `TestMain` function creates all tables, seeds data, and tears down after.

- **Auth integration:** POST /login → verify cookie → GET / with cookie → 200
- **Upload integration:** Request pre-signed URL → simulate S3 upload → mark complete → verify form in DynamoDB
- **Analysis integration:** Mock Bedrock response → trigger analysis → verify version 1 in FieldPlacements table
- **Editor integration:** Save fields → verify new version → save again → verify version incremented → list versions → verify count
- **Revert integration:** Create 3 versions → revert to v1 → verify v4 exists with v1's data and source `revert_from_v1`
- **Customer CRUD integration:** Create → list (count+1) → update → get (verify updated) → delete → list (count-1)
- **Population integration:** Create form + fields + customer → GET download → verify PDF bytes are non-empty and valid PDF header (`%PDF-`)
- **Font integration:** Set form font to Courier 14pt → override one field to 9pt → download PDF → verify (PDF text extraction or byte inspection)

---

#### 7.3 End-to-End Tests (Playwright)

Full browser-based tests against the **live production deployment**. Run after the SAM deploy step completes in GitHub Actions.

**Tech stack:**
- `@playwright/test` (TypeScript)
- Located in `e2e/` directory with its own `package.json`
- `playwright.config.ts` reads `BASE_URL` from environment (set to the API Gateway URL output by SAM deploy)
- Tests run in **Chromium** (headless) in CI, all three browsers locally

**Shared helpers** (`e2e/helpers/auth.ts`)
- `login(page)` — navigates to /login, fills email/password, submits, waits for dashboard
- `storageState` — Playwright auth state file so tests reuse the session without re-logging in for every test

**Test fixtures** (`e2e/fixtures/`)
- `sample-form.pdf` — a simple 1-page PDF with labeled fields (Name, Address, etc.)
- `sample-form.png` — a PNG image of the same form

**Test suites:**

`smoke.spec.ts` — **Quick sanity check (runs first)**
- Navigate to BASE_URL → redirected to /login
- Log in with seed credentials → dashboard loads
- Dashboard shows navigation links (Forms, Customers, Logout)

`auth.spec.ts` — **Authentication flows**
- Login with valid credentials → lands on dashboard
- Login with wrong password → error message displayed, stays on login page
- Login with non-existent email → same error message (no user enumeration)
- Access /forms without logging in → redirected to /login
- Logout → redirected to login, accessing /forms again redirects to login
- Session persistence → login, close tab, open new tab to BASE_URL → still authenticated

`forms.spec.ts` — **Form upload and library management**
- Upload a PDF via the upload flow:
  1. Click "Upload New Form"
  2. Select `sample-form.pdf`
  3. Wait for upload to complete
  4. Verify form appears in the form library
- Upload a PNG image → same flow, verify it appears
- Rename a form → verify new name in library
- Delete a form → verify it disappears from the library
- Empty state: delete all forms → verify "No forms" message displayed

`editor.spec.ts` — **Form editor interactions**
- Open editor for an uploaded form → form image visible
- AI analysis completes → field markers appear on the form
- Drag a field marker → save → reload page → field is in new position
- Resize a field marker → save → reload → field has new dimensions
- Add a new field → verify it appears in the fields panel and on the form
- Remove a field → verify it disappears
- Change form-level font family dropdown → verify selection persists after save
- Change form-level font size → verify persists after save
- Select a field → set field-level font size override → save → reload → override still set
- Clear a field font override → verify it reverts to showing form default

`versioning.spec.ts` — **Version history**
- After AI analysis: version history shows "v1" with source "AI Analysis"
- Edit and save → version history shows "v2" with source "Manual Edit"
- Edit and save again → "v3" visible
- Click on v1 in history → field positions match the original AI placement
- Click "Revert to v1" → new v4 created, field positions match v1
- Version list shows v1, v2, v3, v4 in order

`customers.spec.ts` — **Customer management**
- Customer list shows all 5 seeded customers
- View a customer → all fields displayed (name, business, address, etc.)
- Create a new customer → fill all fields → save → appears in list
- Edit a customer → change business name → save → verify updated in list
- Delete a customer → confirm deletion → verify removed from list

`populate.spec.ts` — **Form population and download**
- Select a form and a customer → preview page shows form with customer data overlaid
- Verify customer name appears on the preview at approximately the right location
- Click Download → PDF file downloads (verify the download event fires and file size > 0)
- Populate with a different customer → verify different name shown in preview

`settings.spec.ts` — **Standard fields configuration**
- Settings page shows all 7 default fields
- Add a custom field (e.g., "Email") → verify it appears in the list
- Remove a field → verify it disappears
- Rename a field → verify new name persists after page reload

**Playwright configuration highlights:**
```typescript
// e2e/playwright.config.ts
export default defineConfig({
  testDir: './tests',
  timeout: 60_000,           // 60s per test (Lambda cold starts)
  retries: 1,                // retry once for flakiness (cold starts, network)
  use: {
    baseURL: process.env.BASE_URL,
    screenshot: 'only-on-failure',
    trace: 'on-first-retry',
    video: 'on-first-retry',
  },
  projects: [
    { name: 'chromium', use: { ...devices['Desktop Chrome'] } },
  ],
  reporter: [['html', { open: 'never' }], ['github']],
});
```

---

#### 7.4 GitHub Actions — Test Pipelines

**Unit + Integration tests** run in the existing `deploy.yml` **before** `sam build`:

```yaml
- name: Run unit tests
  run: go test -race -coverprofile=coverage.out ./...

- name: Check coverage threshold
  run: |
    COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | tr -d '%')
    echo "Coverage: ${COVERAGE}%"
    if (( $(echo "$COVERAGE < 80" | bc -l) )); then
      echo "Coverage ${COVERAGE}% is below 80% threshold"
      exit 1
    fi

- name: Run integration tests
  run: |
    docker run -d -p 8000:8000 amazon/dynamodb-local
    go test -tags=integration -race ./...
```

**E2E tests** run in a **separate job** in `deploy.yml` that depends on the deploy job:

```yaml
e2e:
  needs: deploy
  runs-on: ubuntu-latest
  steps:
    - uses: actions/checkout@v4
    - uses: actions/setup-node@v4
      with:
        node-version: '20'
    - name: Install Playwright
      working-directory: e2e
      run: |
        npm ci
        npx playwright install --with-deps chromium
    - name: Run E2E tests
      working-directory: e2e
      env:
        BASE_URL: ${{ needs.deploy.outputs.api_url }}
        TEST_USER_EMAIL: "proffitt.jeremy@gmail.com"
        TEST_USER_PASSWORD: ${{ secrets.SEED_USER_PASSWORD }}
      run: npx playwright test
    - name: Upload test artifacts
      if: failure()
      uses: actions/upload-artifact@v4
      with:
        name: playwright-report
        path: e2e/playwright-report/
        retention-days: 7
```

---

#### 7.5 Test Data Management

- E2E tests create their own test data (forms, customers) and clean up after themselves
- Each test suite uses unique names/prefixes (e.g., `E2E-Test-Form-{timestamp}`) to avoid collisions
- A global teardown hook deletes any test data left behind by failed tests
- The seed user and seed customers are never modified by E2E tests — tests create separate test customers

---

#### 7.6 UI Polish

- Error handling and user feedback (toast notifications)
- Loading states during AI analysis and file upload
- Empty states (no forms, no customers)
- Responsive layout (functional on tablet+)
- Version history visual indicators

---

## Implementation Order & Dependencies

```
Step 1: Scaffolding & Infrastructure (SAM template, DynamoDB tables, S3, GH Actions)
  │
  ├──> Step 1.5: UI Mockups [HUMAN-IN-THE-LOOP GATE]
  │     ├── UI Engineer + Developer build 9 HTML mockups (parallel)
  │     ├── QA Engineer reviews all mockups
  │     └── USER APPROVAL REQUIRED before proceeding
  │
  ├──> Step 2: Authentication (login, sessions, middleware)
  │       │ (login page implements approved mockup)
  │       │
  │       └──> Step 3: Core Backend (routes, templates, DynamoDB stores, S3 pre-signed URLs)
  │               │ (all pages implement approved mockups)
  │               │
  │               ├──> Step 4: AI Integration (Bedrock analysis → version 1)
  │               │       │
  │               │       └──> Step 5: Form Editor (drag/drop, font config, versioning UI)
  │               │               │ (editor implements approved editor mockup)
  │               │               │
  │               │               └──> Step 6: Population & Download (PDF with fonts)
  │               │
  │               └──> Step 7: Testing
  │                       ├──> 7.1-7.2: Unit + Integration tests (TDD — written BEFORE code at each step)
  │                       ├──> 7.3: Playwright E2E tests (written at steps 5-6, run after deploy)
  │                       └──> 7.4: CI pipeline wiring (after first deploy)
  │
  └──> GitHub Actions pipeline (validate early with hello-world deploy + seed)
```

---

## Potential Issues & Shortcomings

### Significant Concerns

1. **AI Accuracy on Complex Forms (POC Accepted)**
   - Forms with unusual layouts, dense text, or poor scan quality may produce inaccurate field placements
   - The AI may struggle with forms that have multiple similar fields (e.g., "Home Address" vs "Mailing Address")
   - **Status:** Accepted for POC. The manual editor + version history are the safety net

2. **PDF Rendering Fidelity**
   - Overlaying text on existing PDFs while preserving formatting is non-trivial
   - Font family support is limited to standard PDF fonts (Helvetica, Courier, Times-Roman) unless custom fonts are embedded
   - Text may not perfectly align with printed form fields
   - **Mitigation:** Font and size are now configurable per-form and per-field; start with standard PDF fonts; iterate

3. **Cold Start with DynamoDB Connections**
   - Lambda cold start now includes initializing DynamoDB and S3 clients
   - **Mitigation:** Go SDK v2 is lightweight; cold start should remain under 3s. Use lazy initialization where possible

4. **DynamoDB Version Accumulation**
   - Heavy editing of forms creates many version records
   - No automatic cleanup mechanism
   - **Mitigation:** Acceptable for POC. Add TTL or retention policy later if needed

5. **Single Seed User Security**
   - The URL and known credentials could allow unauthorized access
   - **Mitigation:** Change password after initial deploy; restrict API Gateway to IP allowlist or VPN for non-demo use

### Minor Concerns

6. **No Concurrent Edit Protection**
   - Multiple sessions could edit the same form simultaneously, with last-write-wins
   - **Mitigation:** Acceptable for POC single-user scenario

7. **No Form Type Detection**
   - The system treats all forms identically; doesn't recognize that a W-9 is a W-9
   - **Future:** AI could identify the form type and suggest field mappings from a known template library

8. **Limited File Format Support**
   - Supports PDF and common image formats only (no DOCX, etc.)
   - **Mitigation:** Document supported formats clearly

9. **No Customer Data Validation Against Field Size**
   - Long business names or addresses may overflow field boundaries
   - **Mitigation:** Auto-shrink text is a future enhancement; for now users can increase field size

10. **Pre-Signed URL Expiry**
    - If a user takes too long to upload after requesting the URL, it expires (5 min)
    - **Mitigation:** Frontend can request a new URL if the upload fails

11. **Standard PDF Font Limitations**
    - Only Helvetica, Courier, and Times-Roman families available without custom font embedding
    - **Mitigation:** Sufficient for POC; custom font upload is a future enhancement

---

## AWS Resources & Estimated Costs (Phase 1 / Dev)

| Resource | Usage Estimate | Monthly Cost |
|----------|---------------|-------------|
| Lambda | ~10,000 invocations, 512MB, avg 500ms | < $1 |
| API Gateway | ~10,000 requests | < $1 |
| S3 | < 1GB storage, ~1,000 requests | < $1 |
| DynamoDB | On-demand, < 1GB, ~50K RCU/WCU | < $2 |
| Bedrock (Sonnet) | ~100 form analyses, ~1000 input tokens + image each | ~$5-15 |
| **Total** | | **~$10-20/month** |

---

## GitHub Secrets Required

| Secret | Description |
|--------|-------------|
| `AWS_ACCESS_KEY_ID` | IAM user access key for deployment |
| `AWS_SECRET_ACCESS_KEY` | IAM user secret key for deployment |
| `AWS_REGION` | Deployment region (default: `us-east-1`) |
| `SAM_S3_BUCKET` | S3 bucket for SAM deployment artifacts |
| `JWT_SECRET` | Secret key for signing session tokens |
| `SEED_USER_PASSWORD` | Seed user password (used by E2E tests to log in) |

---

## Definition of Done (Phase 1)

### Functionality
- [ ] User can log in with email/password
- [ ] Unauthenticated requests redirect to login
- [ ] User can upload a PDF/image form via pre-signed S3 URL
- [ ] AI analyzes the form and suggests field placements (stored as version 1)
- [ ] User can view and edit field placements in the form editor
- [ ] User can set form-level font family and size
- [ ] User can override font settings per individual field
- [ ] Each save creates a new version; version history is browsable
- [ ] User can revert to any previous version
- [ ] User can select a form and a customer to auto-populate
- [ ] User can download the populated form as a PDF (with correct fonts)
- [ ] Customer data is stored in DynamoDB and fully manageable (CRUD)

### Infrastructure
- [ ] GitHub Actions deploys successfully to AWS Lambda via SAM
- [ ] Seed script creates initial user and mock customers
- [ ] Application is accessible via API Gateway URL

### Testing
- [ ] Go unit test coverage >= 80%
- [ ] All Go unit tests pass (`go test ./...`)
- [ ] All Go integration tests pass (`go test -tags=integration ./...`)
- [ ] Playwright smoke test passes against production URL
- [ ] Playwright auth suite passes (login, logout, session, redirect)
- [ ] Playwright forms suite passes (upload, library, delete)
- [ ] Playwright editor suite passes (drag, resize, add, remove, font config)
- [ ] Playwright versioning suite passes (create, list, revert)
- [ ] Playwright customers suite passes (CRUD)
- [ ] Playwright populate suite passes (preview, download)
- [ ] Playwright test artifacts (screenshots, traces) uploaded on failure
- [ ] E2E tests clean up their own test data
