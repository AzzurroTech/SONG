SONG – Server‑Side Web‑Component Builder

Version: 1.0 (July 23 2025)
Author: Azzurro Technology
Table of Contents

    Project Overview
    Features
    Prerequisites
    Installation & Build
    Running the Server
    Authentication
    API Endpoints
    Static UI Pages
    Data Flow & Component Lifecycle
    Extending / Persisting Data
    Security Considerations
    License

Project Overview

SONG (Server Of New Graphics) is a lightweight Go server that lets authenticated users:

    Define custom web components using standard HTML tags and optional JavaScript.
    Assemble pages from those components.
    Navigate the resulting pages like a normal website.
    Store arbitrary JavaScript snippets without executing them on the server.

All functionality is built exclusively with the Go standard library and vanilla client‑side JavaScript—no external dependencies, no GitHub imports.
Features
#	Capability	Implementation Details
1	Component creation	POST /api/components – stores HTML + optional JS, returns generated ID.
2	Page assembly	POST /api/pages – defines a slug, title, description, and ordered component IDs.
3	Normal navigation	Pages rendered at /pages/{slug} with a simple navigation bar.
4	JavaScript storage (blocked execution)	POST /api/js saves raw source; UI displays it in a read‑only <pre>/<textarea>.
5	Component & page editing	UI forms load existing data (by ID or slug) and re‑POST to update.
6	Basic authentication	All /api/* routes wrapped in a middleware that checks HTTP Basic Auth.
7	Zero external libraries	Pure Go standard library (net/http, encoding/json, sync, etc.) and plain HTML/JS.
Prerequisites

    Go ≥ 1.22 (any recent version supporting modules).
    A modern browser (Chrome, Firefox, Edge, Safari) for the UI.

No other software is required.
Installation & Build

# Clone (or copy) the repository structure locally
git clone https://github.com/AzzurroTech/SONG.git   # optional – you can also just copy the files
cd SONG

# Initialise a Go module (if not already done)
go mod init song

# Build the binary (optional – you can also run directly)
go build -o song-server main.go

All source files are:

song/
├─ main.go                # Go server implementation
└─ static/
   ├─ index.html          # Login screen
   ├─ dashboard.html      # Main control panel
   ├─ component.html      # Component editor
   ├─ page.html           # Page builder
   └─ editjs.html         # JavaScript snippet viewer

Running the Server

# Run the compiled binary or use `go run`
./song-server            # or: go run main.go

The server listens on http://localhost:8080.

Open that URL in a browser. The first page (/) is the login screen.
Default credentials (replace before production):

Username: admin
Password: changeme

After successful login you are redirected to /ui/dashboard.html.
Authentication

All API routes (/api/*) require HTTP Basic Authentication.
The middleware validates against the constants defined in main.go:

const (
    authUser = "admin"
    authPass = "changeme"
)

Replace these values with environment‑derived secrets or a proper password store for real deployments.

The UI stores the Base64‑encoded username:password string in sessionStorage and adds it to every request’s Authorization header.
API Endpoints
Method	Path	Description	Request Body (JSON)	Response
GET	/api/components	List all stored components	—	[{id, html, js}]
POST	/api/components	Create a new component	{ "html": "...", "js": "..." }	{ "id": "...", "html": "...", "js": "..." }
GET	/api/pages/{slug}	Retrieve a page definition (internal use)	—	{ "slug": "...", "title": "...", "components": [...], "description": "..." }
POST	/api/pages	Create a new page	{ "slug":"home", "title":"Home", "components":["id1","id2"], "description":"…" }	Same payload echoed
PUT	/api/pages/{slug}	Update an existing page	Same as POST payload (slug taken from URL)	Updated page JSON
POST	/api/js	Store a JavaScript snippet (read‑only)	{ "key":"myUtil", "code":"function foo(){...}" }	{ "status":"saved" }
GET	/api/js/{key}	Retrieve a stored snippet	—	{ "code":"function foo(){...}" }

All API responses have Content-Type: application/json.
Static UI Pages
URL	Purpose
/ui/index.html	Login screen (basic auth).
/ui/dashboard.html	Central hub linking to component, page, and JS editors.
/ui/component.html	Form to create or edit a component (HTML + optional JS).
/ui/page.html	Form to create or edit a page (slug, title, component ordering).
/ui/editjs.html	Viewer for stored JavaScript snippets (read‑only).
/pages/{slug}	Rendered page that includes the custom elements defined by the selected components.

All UI files are pure HTML/CSS/vanilla JavaScript and reference no external CDNs.
Data Flow & Component Lifecycle

    Create Component – UI sends POST /api/components. Server generates a short ID, stores the HTML/JS pair, and returns the ID.
    Register Custom Element – The client‑side code (inside component.html) registers a custom element named comp-<first‑8‑chars-of-id> that injects the stored HTML and optionally runs the supplied JS in its class.
    Create Page – UI posts to /api/pages with a slug and an ordered list of component IDs.
    Render Page – When a visitor navigates to /pages/<slug>, the server builds a simple HTML document: a navigation bar + a sequence of <comp-xxxxxxx> tags. Browser loads the component definitions (already cached from previous interactions) and displays the composed page.
    JavaScript Snippets – Stored via /api/js; the UI only displays them, never executes them, satisfying the “block code execution” requirement.

All data lives in in‑memory maps (compStore, pageStore, jsStore). Restarting the server clears the data.
Extending / Persisting Data

To move beyond volatile storage, replace the map‑based stores with a lightweight database (e.g., SQLite) using only the Go standard library’s database/sql package and the built‑in sqlite3 driver (available in the standard distribution). Example steps:

import (
    "database/sql"
    _ "modernc.org/sqlite" // pure Go driver, no CGO
)

db, _ := sql.Open("sqlite", "./song.db")
// Create tables for components, pages, js_snippets
// Implement CRUD functions that operate on db instead of the maps

For production‑grade authentication, read credentials from environment variables and hash passwords with golang.org/x/crypto/bcrypt (still a standard‑library‑compatible approach).
Security Considerations

    Basic Auth – Transmit over HTTPS in any non‑local deployment.
    Password Management – Do not ship the default credentials; replace them with strong, secret values.
    Input Validation – The prototype trusts incoming HTML/JS; in a real product you’d sanitize or sandbox user‑provided markup to prevent XSS or injection attacks.
    Rate Limiting – Not implemented; consider adding a simple token bucket if exposing publicly.
    CSRF – Since the UI stores credentials in sessionStorage and sends them via Authorization header, standard CSRF vectors are mitigated, but you may still want to implement same‑site cookies or anti‑CSRF tokens for added safety.

License

The code in this repository is released under the MIT License. Feel free to modify, redistribute, or incorporate it into your own projects, provided the license terms are retained.

Enjoy building with SONG! If you encounter issues or have feature ideas, open an issue on the repository or contact Azzurro Technology support.
