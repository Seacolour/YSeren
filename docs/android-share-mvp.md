# Android Media Source

- Status: Phase 3 complete; second-generation UI, API convergence, emulator QA, and physical-device LAN playback verified
- Last updated: 2026-07-24

## Product Goal

Make YSeren work in the opposite direction of the desktop build:

- desktop YSeren: expose PC media to the LAN
- Android YSeren: expose Android media to the LAN

The focus is still only one thing: turn local media into LAN-accessible HTTP
sources.

## Current Product Scope

- User picks one directory through SAF.
- App stores the directory permission.
- App runs a foreground HTTP share service.
- App exposes the source under the stable contract name `android`.
- Compose provides the same “Share / Sources / Settings” information
  architecture as Desktop, using mobile bottom navigation.
- External clients can use:
  - browser access
  - direct HTTP file URLs
  - M3U playlist import
  - third-party video players

## Current Endpoints

- `/`
- `/api/status`
- `/api/version`
- `/api/tree`
- `/api/tree?q=...`
- `/api/tree?path=...`
- `/playlist.m3u`
- `/stream/android/<relative-path>` (canonical)
- `/stream/<relative-path>` (legacy compatibility)

The standard `/api/tree` response is `root -> android -> media`, so the shared
Svelte Web Player can browse Android and Go servers without platform-specific
branches. `size` is expressed in bytes, `modTime` and `generatedAt` are Unix
seconds, and files contain `source`, `url`, MIME, and media-type fields. Raw SAF
URIs are intentionally omitted from LAN status responses.

## Streaming Behavior

- Full `GET` returns `200` with the exact content length.
- `HEAD` returns the same key headers without a response body.
- Bounded, open-ended, and suffix ranges return `206`.
- Malformed, multiple, and out-of-range requests return `416` with
  `Content-Range: bytes */<total>`.
- Only `GET` and `HEAD` are allowed for stream URLs.
- SAF input streams are closed after completion, truncation, or error.

## Known Tradeoffs

- No transcoding
- No authentication yet
- Directory access is limited to user-selected SAF trees
- SAF stream skipping is not yet optimized for heavy random seeks
- Service is foreground-first, because Android background execution is restrictive

## Emulator Acceptance Record

The 2026-07-24 validation used a LeiDian Android 9/API 28 emulator and a selected
`Pictures/videos` directory containing three MP4 files plus one ZIP archive.

- Directory permission, readable-path presentation, replace/remove/rescan, and
  a three-video/zero-audio scan result were verified.
- Start/stop, foreground notification state, local browser opening, and live
  port changes `1479 -> 1480 -> 1479` were verified.
- Full/partial/legacy HTTP requests and `416` behavior were exercised against
  the running app.
- A Windows browser reached the service through ADB port forwarding, opened the
  shared tree, loaded `bear.mp4`, and advanced playback with no media or console
  errors; media requests returned `206`.
- `:app:testDebugUnitTest`, `:app:assembleDebug`, `:app:lintVitalRelease`, and
  `:app:assembleRelease` passed.

LeiDian's NAT address was not directly reachable from Windows, so the emulator
result was not counted as physical-device LAN evidence.

## Physical-device Acceptance Record

The final Phase 3 gate was completed on 2026-07-24 with a physical Android
device and Windows Chrome on the same Wi-Fi network.

- The app shared `Internal storage / Movies` and indexed 24 media files.
- The foreground service advertised `http://192.168.50.171:1479/` as a LAN URL.
- Windows Chrome reached that URL without ADB forwarding, browsed the standard
  `root -> android` tree, opened a media file, and played it successfully.
- This verifies the complete product path: Android media remains on the phone
  while another device browses and plays it directly over the LAN.

## Recommended Next Steps

1. Improve address prioritization and labeling when Wi-Fi, cellular, or VPN interfaces coexist
2. Add mDNS / Bonjour service discovery
3. Add optional access token protection
4. Add better range IO using seekable file descriptors where providers support it
5. Add media library indexing cache for large folders
6. Revisit multi-source Android support after the single-source workflow is stable
