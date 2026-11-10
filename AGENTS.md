# LocalSend on NSGo

LocalSend on NSGo is a cross-platform LocalSend client built with NSGo. Read [ARCHITECTURE.md](ARCHITECTURE.md) before coding, then the code around the part you change.

## Principles

- Keep `internal/localsend` free of NSGo imports and of the screen's state. Protocol work belongs there; the app only chooses what to show and what to remember.
- The engine goroutine owns signals. Callbacks from the receiver, client and discovery post through `cx.Post`; read values they share through atomics or copies.
- LocalSend compatibility comes first. Check the [protocol](https://github.com/localsend/protocol) and, when a change touches identity, TLS or discovery, test against the official CLI or app.
- Every transfer pins the peer's fingerprint. Discovery may accept an unknown peer, but request metadata and bytes may not.
- A received file never leaves its receive folder. Sanitize names, keep a part file until the write and its checksum succeed, and remove what failed.
- Use `base` and `ui` for components and the theme for color; no hand-drawn platform widgets.
- The app's words come from `locales/*.yaml`, in English and Simplified Chinese, never string literals in components.

## Working with the user

- Never create sub-Agents or Git worktrees; work directly in the current checkout.
- Do not edit the `nsgo/` reference checkout. If NSGo itself needs a change, report it.
- Commit or push only when the user asks; Neorio lands the branch.
- When a change adds a program or changes how the app installs, starts or builds, update the workspace's Neorio `config.yaml` in the same task.
- Answer the user in the language of their latest message. Code, comments, commits and documents are English.

## Verification

- `go test ./...` for the protocol and headless app tests.
- `go tool nsgo test -target macos|windows|linux|ios_simulator|android` for native windows and input. Use the workspace's simulator, Windows and Linux machines, and uninstall an old Android app before installing a new one.
- Test sending and receiving both ways, in HTTPS and HTTP modes, and check the receive folder after a transfer.

## Documentation

- Update a document only when a fact its reader needs changes.
- Avoid labels and descriptions that restate a heading or a value; let the layout explain.
- Write only what code and comments cannot show: purpose, boundaries and reasons, not names, fields, defaults or step-by-step behavior.
- Keep each document within its budget; check its size after editing and trim it before handing off when it is over.
- Record completed user-visible outcomes as one-clause `Unreleased` bullets in [CHANGELOG.md](CHANGELOG.md), ordered by type and merged by capability.
- Repair documents you touch instead of only appending to them.
- Keep [README.md](README.md) and [README.zh-CN.md](README.zh-CN.md) equivalent.
- Keep documentation on the code branch and reconcile it during merges.
