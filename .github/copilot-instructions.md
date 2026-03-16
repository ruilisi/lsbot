# lsbot Project Instructions

## Configuration Resolution Order

All configuration in lsbot follows a strict three-tier priority: **CLI flags > environment variables > config file (`~/.lsbot.yaml`)**. See [CONFIGURATION.md](CONFIGURATION.md) for full details.

When adding new configurable parameters, you MUST follow this pattern:
1. Accept a CLI flag (highest priority)
2. Fall back to an environment variable
3. Fall back to `~/.lsbot.yaml` config file (lowest priority)

Never skip a tier or change the resolution order. This is a core architectural invariant.

## Documentation Placement

- **Project-wide docs** (applicable to all platforms/modes) go at the repo root alongside `README.md` (e.g., `CONFIGURATION.md`)
- **Platform-specific or feature-specific docs** go in `docs/` (e.g., `docs/wecom-integration.md`, `docs/browser-automation.md`)

## Browser Automation

### Peekaboo (macOS UI Automation)

Use `peekaboo` for macOS UI automation: capture/inspect screens, target UI elements, drive input, and manage apps/windows/menus. Commands share a snapshot cache and support `--json`/`-j` for scripting.

**Quickstart:**

```bash
peekaboo permissions
peekaboo list apps --json
peekaboo see --annotate --path /tmp/peekaboo-see.png
peekaboo click --on B1
peekaboo type "Hello" --return
```

**Core commands:**

- `image`: capture screenshots (screen/window/menu bar regions)
- `see`: annotated UI maps, snapshot IDs, optional analysis
- `list`: apps, windows, screens, menubar, permissions
- `capture`: live capture or video ingest + frame extraction

**Interaction:**

- `click`: target by ID/query/coords with smart waits
- `drag`: drag & drop across elements/coords/Dock
- `hotkey`: modifier combos like `cmd,shift,t`
- `type`: text + control keys (`--clear`, delays)
- `press`: special-key sequences with repeats
- `scroll`: directional scrolling (targeted + smooth)
- `swipe`: gesture-style drags between targets
- `paste`: set clipboard -> paste -> restore
- `move`: cursor positioning with optional smoothing

**System:**

- `app`: launch/quit/relaunch/hide/unhide/switch/list apps
- `window`: close/minimize/maximize/move/resize/focus/list
- `menu`: click/list application menus + menu extras
- `menubar`: list/click status bar items
- `dock`: launch/right-click/hide/show/list Dock items
- `dialog`: click/input/file/dismiss/list system dialogs
- `clipboard`: read/write clipboard (text/images/files)
- `open`: enhanced `open` with app targeting + JSON payloads
- `space`: list/switch/move-window (Spaces)

**Common targeting parameters:**

- App/window: `--app`, `--pid`, `--window-title`, `--window-id`, `--window-index`
- Snapshot targeting: `--snapshot` (ID from `see`; defaults to latest)
- Element/coords: `--on`/`--id` (element ID), `--coords x,y`

**Workflow: see -> click -> type (most reliable):**

```bash
peekaboo see --app Safari --window-title "Login" --annotate --path /tmp/see.png
peekaboo click --on B3 --app Safari
peekaboo type "user@example.com" --app Safari
peekaboo press tab --count 1 --app Safari
peekaboo type "supersecret" --app Safari --return
```

**Notes:**
- Requires Screen Recording + Accessibility permissions.
- Use `peekaboo see --annotate` to identify targets before clicking.

### Playwright Browser Automation

For web browser automation, use Playwright via CDP (Chrome DevTools Protocol) to control Chromium-based browsers.

**Key concepts:**
- Use a **dedicated browser profile** isolated from personal browsing
- Control tabs deterministically by `targetId`
- Use **snapshots** (accessibility tree) to identify elements, then **act** on refs
- Refs are not stable across navigations; re-snapshot after page changes

**Snapshot types:**
- **AI snapshot** (numeric refs): default, text-based with numeric refs for actions
- **Role snapshot** (role refs like `e12`): interactive element list with role-based refs

**Action flow:**
1. Take a snapshot to get element refs
2. Use refs to click/type/drag/select
3. Re-snapshot if the page changed

## Skills

### /review-code - Code Review After Changes

Review code changes in the current branch compared to main. Provide structured feedback on correctness, design, performance, security, and test coverage.

### /github - GitHub CLI Reference

Use the `gh` CLI to interact with GitHub. Always specify `--repo owner/repo` when not in a git directory.

**PR checks and CI:**

```bash
gh pr checks 55 --repo owner/repo
gh run list --repo owner/repo --limit 10
gh run view <run-id> --repo owner/repo
gh run view <run-id> --repo owner/repo --log-failed
```

**API for advanced queries:**

```bash
gh api repos/owner/repo/pulls/55 --jq '.title, .state, .user.login'
```

**JSON output with filtering:**

```bash
gh issue list --repo owner/repo --json number,title --jq '.[] | "\(.number): \(.title)"'
```

### /mcporter - MCP Server Management

Use `mcporter` to work with MCP servers directly.

**Quick start:**

```bash
mcporter list
mcporter list <server> --schema
mcporter call <server.tool> key=value
```

**Call tools:**

- Selector: `mcporter call linear.list_issues team=ENG limit:5`
- Function syntax: `mcporter call "linear.create_issue(title: \"Bug\")"`
- Full URL: `mcporter call https://api.example.com/mcp.fetch url:https://example.com`
- Stdio: `mcporter call --stdio "bun run ./server.ts" scrape url=https://example.com`
- JSON payload: `mcporter call <server.tool> --args '{"limit":5}'`

**Auth + config:**

- OAuth: `mcporter auth <server | url> [--reset]`
- Config: `mcporter config list|get|add|remove|import|login|logout`

**Daemon:** `mcporter daemon start|status|stop|restart`

**Codegen:**

- CLI: `mcporter generate-cli --server <name>`
- TS: `mcporter emit-ts <server> --mode client|types`