# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project aims to follow [Semantic Versioning](https://semver.org/).

## [Unreleased]

### Added

- GitHub Actions workflows for CI validation and tagged release builds.
- Multi-platform release packaging plan for GitHub Releases.
- `CONTRIBUTING.md`, `SECURITY.md`, issue templates, and a PR template.
- `yseren.example.yaml` as the public config template.
- Apache-2.0 license metadata.

### Changed

- Repository hygiene for open-source publishing by ignoring local runtime config and IDE files.
- README now includes release, package, and contributor-facing guidance.

### Fixed

- Search results now render from the latest filtered tree instead of stale navigation state.
- Stream URLs now safely encode source names and path segments.
- Custom audio extensions can be configured without being misclassified as video playback.
