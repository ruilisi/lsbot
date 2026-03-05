# Skills

Skills are modular capability packs that teach lingti-bot how to use external tools. Each skill is a directory containing a `SKILL.md` file — a markdown document with YAML frontmatter that declares requirements, metadata, and instructions.

## Quick Start

```bash
# List all discovered skills
lingti-bot skills

# See what's ready vs missing
lingti-bot skills check

# Get details on a specific skill
lingti-bot skills info github

# Disable / enable a skill
lingti-bot skills disable weather
lingti-bot skills enable weather
```

## CLI Reference

### `lingti-bot skills` / `lingti-bot skills list`

List all discovered skills with their eligibility status.

```
lingti-bot skills list [flags]
```

| Flag | Description |
|------|-------------|
| `--eligible` | Show only ready-to-use skills |
| `-v, --verbose` | Show missing requirements column |
| `--json` | Output as JSON |

Example:

```
$ lingti-bot skills list -v

Skills (6/8 ready)

  Status       Skill            Description                          Source      Missing
  ✗ missing    🔐 1password     Set up and use 1Password CLI (op)... bundled     bins: op
  ✓ ready      🎮 discord       Use when you need to control Disc... bundled
  ✓ ready      🐙 github        Interact with GitHub using the g... bundled
  ✗ missing    💎 obsidian       Work with Obsidian vaults (plain... bundled     bins: obsidian-cli
  ✓ ready      👀 peekaboo      Capture and automate macOS UI wit... bundled
  ✓ ready      💬 slack          Use when you need to control Slac... bundled
  ✓ ready      🧵 tmux          Remote-control tmux sessions for... bundled
  ✓ ready      🌤️ weather       Get current weather and forecasts... bundled
```

### `lingti-bot skills info <name>`

Show detailed information for a single skill — description, source path, requirements with per-item status, and install hints.

```
$ lingti-bot skills info 1password

🔐 1password ✗ Missing requirements

Set up and use 1Password CLI (op). Use when installing the CLI,
enabling desktop app integration, signing in, or reading/injecting secrets.

Details:
  Source:   bundled
  Path:     ~/Projects/lingti-bot/bundled-skills/1password/SKILL.md
  Homepage: https://developer.1password.com/docs/cli/get-started/

Requirements:
  Binaries:  ✗ op

Install options:
  → Install 1Password CLI (brew)
```

### `lingti-bot skills check`

Summary view — counts by status, lists ready skills and what's missing.

```
$ lingti-bot skills check

Skills Status Check

Total:                  8
✓ Eligible:             6
⏸ Disabled:             0
✗ Missing requirements: 2

Ready to use:
  🎮 discord
  🐙 github
  👀 peekaboo
  💬 slack
  🧵 tmux
  🌤️ weather

Missing requirements:
  🔐 1password (bins: op)
  💎 obsidian (bins: obsidian-cli)
```

### `lingti-bot skills enable <name>`

Re-enable a previously disabled skill. Removes the name from `skills.disabled` in `bot.yaml`.

### `lingti-bot skills disable <name>`

Disable a skill. Adds the name to `skills.disabled` in `bot.yaml`. The skill remains on disk but is excluded from eligibility checks.

### JSON Output

All read commands support `--json` for scripting:

```bash
lingti-bot skills list --json
lingti-bot skills info github --json
lingti-bot skills check --json
```

## Skill Discovery

Skills are loaded from three directories in precedence order. When two directories contain a skill with the same `name`, the higher-precedence source wins.

| Priority | Location | Description |
|----------|----------|-------------|
| 1 (lowest) | `<binary>/../skills/` or `./bundled-skills/` | Bundled skills shipped with the binary |
| 2 | `~/.lingti/skills/` | User-installed (managed) skills |
| 3 (highest) | `./skills/` | Project-specific (workspace) skills |

The bundled directory is resolved automatically:

1. `$LINGTI_BUNDLED_SKILLS_DIR` environment variable (if set)
2. `<executable-dir>/../skills/`
3. `<executable-dir>/skills/`
4. `./bundled-skills/` (development mode)

Additional directories can be configured in `bot.yaml`:

```yaml
skills:
  extra_dirs:
    - /path/to/shared/skills
    - /another/team/skills
```

## Creating a Skill

### 1. Create the directory

```bash
mkdir -p ~/.lingti/skills/my-tool
```

### 2. Write SKILL.md

```yaml
---
name: my-tool
description: "Short description shown in skills list."
homepage: https://example.com/my-tool
metadata:
  emoji: "🔧"
  os: ["darwin", "linux"]
  requires:
    bins: ["my-tool"]
    env: ["MY_TOOL_API_KEY"]
  install:
    - id: brew
      kind: brew
      formula: example/tap/my-tool
      label: "Install my-tool (brew)"
    - id: npm
      kind: npm
      package: my-tool
      label: "Install my-tool (npm)"
---

# My Tool

Instructions for the AI agent on how to use my-tool...

## Common Commands

\`\`\`bash
my-tool list
my-tool create --name "example"
\`\`\`

## Tips

- Always use `--json` flag for parseable output.
- Use `MY_TOOL_API_KEY` for authentication.
```

### 3. Verify

```bash
lingti-bot skills info my-tool
```

## SKILL.md Format

A SKILL.md file has two parts:

1. **YAML frontmatter** (between `---` delimiters) — machine-readable metadata
2. **Markdown body** — instructions for the AI agent

### Frontmatter Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | **yes** | Unique skill identifier |
| `description` | string | **yes** | Short description (shown in list, truncated to ~36 chars) |
| `homepage` | string | no | URL to documentation or project page |

### Metadata Fields

Nested under `metadata:` in the frontmatter.

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `emoji` | string | 📦 | Emoji shown next to skill name |
| `os` | []string | (all) | Allowed operating systems: `darwin`, `linux`, `windows` |
| `always` | bool | false | Skip all gating — always mark as eligible |
| `requires.bins` | []string | [] | Required binaries — **all** must exist in PATH |
| `requires.any_bins` | []string | [] | At least **one** must exist in PATH |
| `requires.env` | []string | [] | Required environment variables — **all** must be set |
| `install` | []InstallSpec | [] | How to install missing requirements |

### InstallSpec Fields

Each entry in the `install` array:

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Unique identifier for this install method |
| `kind` | string | Package manager: `brew`, `apt`, `go`, `npm`, `download` |
| `formula` | string | Homebrew formula (for `kind: brew`) |
| `package` | string | Package name (for `kind: apt` or `kind: npm`) |
| `module` | string | Go module path (for `kind: go`) |
| `url` | string | Download URL (for `kind: download`) |
| `label` | string | Human-readable label shown in `skills info` |
| `bins` | []string | Binaries this method installs |

### OpenClaw Compatibility

Skills from the [OpenClaw](https://github.com/AgeOfAI/openclaw) project work out of the box. The parser automatically handles OpenClaw's nested metadata format:

```yaml
metadata: {"openclaw": {"emoji": "🐙", "requires": {"bins": ["gh"]}}}
```

This is equivalent to the flat format:

```yaml
metadata:
  emoji: "🐙"
  requires:
    bins: ["gh"]
```

## Eligibility Gating

When a skill is discovered, it goes through a series of gates to determine if it's **eligible** (ready to use):

```
1. Not disabled?         → Check skills.disabled in bot.yaml
2. OS matches?           → Check metadata.os against runtime OS
3. Always flag?          → If metadata.always=true, skip remaining gates
4. All bins exist?       → exec.LookPath for each in requires.bins
5. Any bin exists?       → At least one from requires.any_bins
6. All env vars set?     → os.Getenv for each in requires.env
```

A skill that fails any gate is marked as **missing** (with details of what's missing). A skill explicitly listed in `skills.disabled` is marked as **disabled**.

## Configuration

Skills configuration lives in `bot.yaml` under the `skills` key:

```yaml
# bot.yaml
skills:
  # Skills to exclude from eligibility (by name)
  disabled:
    - obsidian
    - 1password

  # Additional directories to scan for skills
  extra_dirs:
    - /Users/shared/team-skills
    - ~/my-extra-skills
```

Config file location:
- **macOS**: `~/Library/Preferences/Lingti/bot.yaml`
- **Linux**: `~/.config/lingti/bot.yaml`
- **Other**: `~/.lingti/bot.yaml`

## Directory Layout

```
~/.lingti/skills/                  # Managed skills (user-installed)
├── my-custom-skill/
│   └── SKILL.md
└── another-skill/
    └── SKILL.md

<project>/skills/                  # Workspace skills (project-specific)
└── project-tool/
    └── SKILL.md

<binary>/../skills/                # Bundled skills (shipped with binary)
├── github/
│   └── SKILL.md
├── peekaboo/
│   └── SKILL.md
├── tmux/
│   └── SKILL.md
└── weather/
    └── SKILL.md
```

## Bundled Skills

Lingti-bot ships with 8 bundled skills:

| Skill | Emoji | Requires | Description |
|-------|-------|----------|-------------|
| `1password` | 🔐 | `op` | 1Password CLI for secrets management |
| `discord` | 🎮 | — | Discord bot control and moderation |
| `github` | 🐙 | `gh` | GitHub CLI for issues, PRs, CI, and API queries |
| `obsidian` | 💎 | `obsidian-cli` | Work with Obsidian vaults |
| `peekaboo` | 👀 | `peekaboo` | macOS UI automation (screenshots, clicks, typing) |
| `slack` | 💬 | — | Slack workspace control |
| `tmux` | 🧵 | `tmux` | Remote-control tmux sessions for interactive CLIs |
| `weather` | 🌤️ | `curl` | Weather forecasts via wttr.in and Open-Meteo |

## Examples

### Override a bundled skill

Create a skill with the same name in `~/.lingti/skills/` to override the bundled version:

```bash
mkdir -p ~/.lingti/skills/github
cat > ~/.lingti/skills/github/SKILL.md << 'EOF'
---
name: github
description: "Custom GitHub workflow with team conventions."
metadata:
  emoji: "🐙"
  requires:
    bins: ["gh"]
---

# GitHub (Custom)

Always use `--repo myorg/myrepo` by default...
EOF
```

The managed version takes precedence over the bundled one.

### Project-specific skill

Add a skill that only applies to the current project:

```bash
mkdir -p ./skills/deploy
cat > ./skills/deploy/SKILL.md << 'EOF'
---
name: deploy
description: "Deploy this project to production."
metadata:
  emoji: "🚀"
  requires:
    bins: ["kubectl", "helm"]
    env: ["KUBECONFIG"]
---

# Deploy

Use helm to deploy to the staging cluster...
EOF
```

### Skill that requires an API key

```yaml
---
name: openai
description: "OpenAI API integration."
metadata:
  emoji: "🤖"
  requires:
    env: ["OPENAI_API_KEY"]
---
```

If `OPENAI_API_KEY` is not set, the skill shows as **missing** with a clear indication of what's needed.
