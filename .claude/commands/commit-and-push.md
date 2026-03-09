---
description: Commit staged changes and create a PR, push on top if PR already exists.
---

# Commit and Push Changes

## Pre-Commit Validation

### 1. Git Status Check

```bash
git branch --show-current
git status --short
git diff --stat
git diff --cached --stat
```

- **NEVER commit to `main`** — all work must be on feature branches
- If on `main`, **STOP immediately** and create a feature branch before doing anything else
- Do not ask "should I proceed on main?" — the answer is always no

### 2. Quality Checks

Run all quality checks before committing. These are **CRITICAL** — do not skip or bypass failures.

```bash
make check
```

This runs: `fmt`, `tidy`, `vet`, `lint`, `test` (with race detection).

- **If `make check` fails**: Fix the issues, do NOT commit broken code
- **If formatting changed files**: Stage the formatted files before committing
- **If `go mod tidy` changed files**: Stage `go.mod` and `go.sum`

### 3. Change Review

Review all changes to understand what's being committed:

```bash
git diff
git diff --cached
git ls-files --others --exclude-standard
```

**Stage files by area** — prefer explicit file paths over `git add .`:

| Directory | Contains |
|-----------|----------|
| `cmd/lazycron/` | Entry point |
| `internal/config/` | App configuration and theme |
| `internal/cron/` | Crontab parsing, scheduling, read/write |
| `internal/gui/` | TUI views, controllers, keybindings, modals |
| `internal/gui/style/` | Colour constants |
| `internal/ssh/` | SSH client and server config |
| `internal/types/` | Shared types and version constant |
| `testdata/` | Crontab test fixtures |
| `.github/workflows/` | CI/CD workflows |

**Never stage**: `.env`, credentials, `.DS_Store`, IDE config, or binary output (`lazycron`, `dist/`)

### 4. README Update Check (MANDATORY)

- **STOP**: You MUST complete this step before proceeding to commit
- **Read `README.md`** to understand current documentation
- **Compare changes against README content** — for each changed file, check if:
  - New commands, features, or functionality were added
  - Installation steps or prerequisites changed
  - Directory structure or file locations changed
  - CLI options, keybindings, or configuration changes need documenting
- **If ANY documentation updates are needed**:
  - Update the README BEFORE creating the commit
  - Stage the README changes along with the other changes
- **If unsure**: Ask the user whether README updates are needed
- **Do NOT skip this step** — documentation drift causes confusion

### 5. CHANGELOG Update (User-Facing Changes)

For commits with these prefixes, update `CHANGELOG.md` before committing:

- `feat:`, `add:`, `update:` — new or changed functionality
- `fix:` — bug fixes
- `breaking:` — breaking changes

**Format:**

```markdown
## [Unreleased]

- Brief description of what changed and why
```

- Skip CHANGELOG updates for `refactor:`, `chore:`, `docs:`, `test:`, `style:` commits
- If `CHANGELOG.md` doesn't exist yet, create it with the above format
- Stage CHANGELOG changes with the rest of the commit

## Commit Process

### Analyse Changes

Review all staged changes:

```bash
git diff --cached
```

Understand the intent: is this a feature, fix, refactor, chore, docs update, or test?

### Generate Commit Message

**Format:** `<type>: <description>`

| Prefix | Usage |
|--------|-------|
| `feat:` | New features or functionality |
| `add:` | Adding new files, commands, or capabilities |
| `update:` | Enhancements to existing features |
| `fix:` | Bug fixes |
| `refactor:` | Code restructuring without behaviour changes |
| `chore:` | Maintenance, config, tooling, dependencies |
| `docs:` | Documentation-only changes |
| `test:` | Adding or updating tests |
| `style:` | Formatting, linting, cosmetic changes |
| `breaking:` | Breaking changes (triggers major version bump) |

**Rules:**
- Lowercase prefix, lowercase description
- Keep the first line under 72 characters
- Use imperative mood ("add search", not "added search")
- Focus on *why*, not *what* — the diff shows what changed
- **NEVER** include `Co-Authored-By` lines or mention Claude/AI

**Commit using a heredoc** (for correct formatting):

```bash
git commit -m "$(cat <<'EOF'
type: description here
EOF
)"
```

### Push

```bash
git push -u origin $(git branch --show-current)
```

## Pull Request Creation

### Check for Existing PR

```bash
gh pr list --head $(git branch --show-current)
```

- **If a PR exists**: Push is sufficient, no new PR needed
- **If no PR exists**: Create one

### Create Pull Request

```bash
gh pr create --title "Short title under 70 chars" --body "$(cat <<'EOF'
## Summary
- Bullet points describing the changes

## Test Plan
- How to verify the changes work
EOF
)"
```

**PR rules:**
- Title under 70 characters
- Summary section with 1-3 bullet points
- Test plan section
- **NEVER** include AI attribution or "Generated with" footers
- Target branch: `main`

## Post-PR Actions

### Verification

After creating or pushing to a PR:

```bash
gh pr checks $(gh pr list --head $(git branch --show-current) --json number -q '.[0].number')
```

- Monitor the CI workflow (lint + test) from `release.yml`
- If checks fail, fix and push again — do not force-push unless asked

## Notes

- **Release automation**: Commits to `main` with recognised prefixes (`feat:`, `fix:`, `breaking:`, `add:`, `update:`) trigger automatic version bumps and GitHub releases via `release.yml`
- **Homebrew**: Releases automatically update the `seanhalberthal/homebrew-tap` formula
- **British English**: Use British spelling in all code, comments, and documentation (colour, behaviour, centre, etc.)
