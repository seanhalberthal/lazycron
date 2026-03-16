## [Unreleased]

- TUI: re-render views on terminal resize so column widths adapt
- Mail: hide status bar badge when all messages are read
- SSH: verify host keys against `~/.ssh/known_hosts` instead of accepting all keys
- SSH: use random AES-GCM encryption key persisted at `~/.config/lazycron/.key`
- SSH: validate server configuration before connecting (host, user, port, auth_type)
- SSH: restrict private key paths to within the user's home directory
- Replace hand-rolled TOML parser with `BurntSushi/toml`

## [0.2.2]

- Friendly cron syntax: human-readable schedule descriptions
- Fix: MarkRead status header duplication
- Close test coverage gaps

## [0.2.1]

- Mail tab: view, read, delete, and manage local and remote mbox mailboxes

## [0.2.0]

- Coloured UI: styled labels, hints bar, help overlay, modal dialogs, and status indicators
- Custom input editor with macOS modifier key support (Opt+Arrow word navigation, Ctrl+W, Ctrl+K)
- Real-time cron expression validation with inline feedback in create/edit modal
- Expression guide panel alongside create/edit modal on wide terminals
- Shift-Tab to cycle backwards through modal fields
- CI workflow for lint and test on non-main branches

## [0.1.0]

- Initial project: TUI for managing cron jobs locally and over SSH
