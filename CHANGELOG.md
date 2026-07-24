# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project aims to follow [Semantic Versioning](https://semver.org/).

## [Unreleased]

### Added

- Go Headless `/api/status` and `/playlist.m3u` endpoints aligned with the Android sharing contract.
- Repeatable Linux Headless smoke coverage for status, tree, playlist, streaming, Range, permissions, symlinks, port conflicts, client interruption, and process signals.

### Changed

- Linux media scans now resolve safe file symlinks, report target metadata, and omit unreadable or escaping entries.
- Go streaming now enforces the same single-range policy as Android, returning `416` for malformed and multiple ranges.
- CI now builds a CGO-disabled Linux Headless ELF and runs the Linux smoke matrix.

## [0.1.2] - 2026-05-18

### Added

- Android Compose/Material3 control console with dashboard, media source, LAN URL, and settings tabs.
- Android app icons, notification icon, Web UI favicon, manifest, and installable PWA-style assets.
- Android foreground share server now packages the YSeren Web UI inside the APK.
- Android `/api/tree` recursive media tree API for the packaged Web UI, while keeping `/api/tree?path=...` diagnostics.
- Android release workflow support for building an APK artifact, with optional signing through repository secrets.

### Changed

- Android app version bumped to `0.1.2` with `versionCode` 2.
- Android share notification now uses the app notification icon.
- Web top bar now uses the YSeren logo and a compact brand header.

## [0.1.1] - 2026-04-10

### Added

- GitHub Actions workflows for CI validation and tagged release builds.
- Multi-platform release packaging plan for GitHub Releases.
- `CONTRIBUTING.md`, `SECURITY.md`, issue templates, and a PR template.
- `yseren.example.yaml` as the public config template.
- Apache-2.0 license metadata.
- Android share MVP scaffold with a foreground HTTP sharing service prototype.

### Changed

- Repository hygiene for open-source publishing by ignoring local runtime config and IDE files.
- README now includes release, package, and contributor-facing guidance.
- README now documents the Android-side media source direction.
- Compressed archives are now ignored instead of being shown or extracted.

### Fixed

- Search results now render from the latest filtered tree instead of stale navigation state.
- Stream URLs now safely encode source names and path segments.
- Custom audio extensions can be configured without being misclassified as video playback.
