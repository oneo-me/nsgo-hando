# Hando architecture

Hando is an NSGo app whose screens are Go components and signals, with the LocalSend protocol in a separate package below it. One app process runs one receiver, one HTTP client and one discovery loop; every screen reads the same signals.

## Code map

| Path | Responsibility |
| --- | --- |
| `main.go` | Entry point and the main window's size and limits. |
| `app.go` | The app's signals and lifetime: settings, favorites, devices, transfers, the receiver node, settings storage, device merging and shared helpers. |
| `nearby.go` | The nearby page: this device, discovery banners, device cards and the empty state. |
| `send.go` | The send dialog, file and folder picks, the prepare/upload client flow and the PIN retry. |
| `receive.go` | The incoming request dialog, quick save, progress and the receive result. |
| `transfers.go` | Active and finished transfer cards, history, cancel and open. |
| `settings.go` | The settings page and the receiver's restart when its network settings change. |
| `ui.go` | The desktop sidebar shell, the phone shell, the header actions, overlays and shared visuals. |
| `menu.go` | The macOS menu bar. |
| `stats.go` | Anonymous usage statistics sent to Umami. |
| `theme.go` | The light and dark palette. |
| `internal/localsend` | Protocol v2.2: identity and mutual TLS, HTTP client, receiver, IPv4 discovery, receive-path planning and file writing. |
| `locales/` | English and Simplified Chinese catalogs and the code `nsgo i18n` writes. |

## Boundaries

- `internal/localsend` imports no NSGo package. Its tests run without an engine, and the same code runs on every target and in manual interop tests.
- The engine goroutine owns every signal. Server, client and discovery callbacks reach the app through `cx.Post`; settings and the PIN that those callbacks read are kept in atomics.
- Discovery registers with peers unpinned, because it learns each fingerprint from the peer's certificate, and accepts only a certificate that is currently valid and self-signed.
- Every transfer pins the fingerprint, so request metadata and file bytes never go to another device answering on that address.
- A LocalSend peer asks for a client certificate naming its own acceptable CAs. The client always presents the app's self-signed certificate through `GetClientCertificate`; Go's default selection would send none.
- Received bytes go below the chosen receive folder. Each file's name is normalized and sanitized, no segment may leave the folder, and a name already taken gets a numeric suffix.
- A file is written as `<name>.part` and renamed after it is complete and, when the sender offered a SHA-256 and verification is on, checked. A failed write, checksum or cancellation leaves no final file and removes the part file.
- One receive session and one upload per file may run at a time; the protocol's session ID and per-file token guard both.
- Settings are the `settings` storage entry; favorites and history are `favorites` and `history`. The TLS identity lives in the app's data folder, not in storage.
- Usage statistics are page views and heartbeats with a random install ID, sent to Umami from release builds on native targets; they never carry device, peer, file or transfer data, and a setting turns them off. Builds of `nsgo dev`, tests and the web send nothing.

## Core flows

**Discovery.** The node announces this device to `224.0.0.167:53317` with a burst, then peers answer with `POST /api/localsend/v2/register`. Received announcements are answered the same way. If nothing answers, the app scans its local IPv4 networks once and probes favorites. A device is merged by fingerprint, so a later answer updates its alias, address and type.

**Send.** The app creates a transfer and a send job, then calls `prepare-upload` with the peer's pinned client. A receiver that asks for a PIN reopens the job through the prompt; a rejection, a busy receiver or a refused PIN ends the transfer with a reason the screen localizes. Accepted files upload one after another, with progress posted to the engine, and a cancel closes the request context and tells the receiver to end the session.

**Receive.** The server emits a prepare request; the app accepts it after the review dialog, or at once for a favorite with Quick Save. The server plans safe destinations, returns one token per accepted file, writes uploads with progress, verifies checksums and ends the session once every accepted file is final. The app mirrors the result into the transfer list and history.

**Receiver restart.** Changing the port, the protocol or checksum verification cancels active sends, stops the old node, and starts a new one with the current settings. Changing the alias or PIN does not restart it, since callbacks read those live.

## Data and state

- `cx.Storage()`: `settings`, `favorites`, the last 50 finished transfers and the statistics' `installID`. Each is rewritten when its signal changes.
- `cx.DataDir()/identity`: the certificate, its key and the HTTP-mode fingerprint. Tests and hosts without a data folder use a temporary folder.
- Received files: the chosen folder, else `Downloads/Hando` on desktops and `DataDir/Received` elsewhere.
- In memory: the discovered devices, the current transfers, the incoming request and the send jobs waiting for a PIN. None is shared with the protocol package.

## Verification

`go test ./...` checks the protocol against another instance over HTTP and mutual TLS, path planning, checksum failure, cancellation, PIN refusal, identity persistence and discovery between two nodes. The app tests run headless and on macOS, Windows, Linux, iOS and Android with `nsgo test -target`, including a fake peer that sends and receives a real file. The client was also checked manually against LocalSend CLI 1.18.2 over mutual TLS; discovery was checked in both directions.

## Subjects

| Document | When to read |
| --- | --- |
| — | No subject documents yet. Add one when a lasting design, such as pairing, outgrows this page. |
