# Doc-Manager: Implementation Plan

## Phase 1 Implementation Roadmap

### Step 1: Project Scaffolding & Infrastructure

**Goal:** Set up the Go project structure, SAM template (with DynamoDB tables), and GitHub Actions pipeline.

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

### Step 7: Testing & Polish

#### 7.1 Unit Tests
- DynamoDB store tests (use local DynamoDB or mocked client)
- Auth middleware tests
- Handler tests (HTTP request/response)
- Bedrock service tests (mock Bedrock responses)
- PDF generation tests (verify text placement and font settings)
- Version management tests (create, list, revert)

#### 7.2 Integration Testing
- Full login flow
- Upload a sample form via pre-signed URL
- Verify AI analysis returns field placements stored as version 1
- Edit fields, save, verify version 2 is created
- Revert to version 1, verify version 3 is created with v1's data
- Populate form with customer data, verify PDF output
- Verify font overrides render correctly

#### 7.3 UI Polish
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
  ├──> Step 2: Authentication (login, sessions, middleware)
  │       │
  │       └──> Step 3: Core Backend (routes, templates, DynamoDB stores, S3 pre-signed URLs)
  │               │
  │               ├──> Step 4: AI Integration (Bedrock analysis → version 1)
  │               │       │
  │               │       └──> Step 5: Form Editor (drag/drop, font config, versioning UI)
  │               │               │
  │               │               └──> Step 6: Population & Download (PDF with fonts)
  │               │
  │               └──> Step 7: Testing (runs in parallel with steps 4-6)
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

---

## Definition of Done (Phase 1)

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
- [ ] GitHub Actions deploys successfully to AWS Lambda via SAM
- [ ] Seed script creates initial user and mock customers
- [ ] Application is accessible via API Gateway URL
- [ ] Basic error handling and user feedback throughout
