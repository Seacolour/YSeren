# Android Share MVP

## Product Goal

Make YSeren work in the opposite direction of the desktop build:

- desktop YSeren: expose PC media to the LAN
- Android YSeren: expose Android media to the LAN

The focus is still only one thing: turn local media into LAN-accessible HTTP
sources.

## First Implementation Scope

- User picks one directory through SAF.
- App stores the directory permission.
- App runs a foreground HTTP share service.
- External clients can use:
  - browser access
  - direct HTTP file URLs
  - M3U playlist import
  - third-party video players

## Current Endpoints

- `/`
- `/api/status`
- `/api/tree?path=...`
- `/playlist.m3u`
- `/stream/<relative-path>`

## Known Tradeoffs

- No transcoding
- No authentication yet
- Directory access is limited to user-selected SAF trees
- Range support is MVP-grade, not yet optimized for heavy random seeks
- Service is foreground-first, because Android background execution is restrictive

## Recommended Next Steps

1. Add mDNS / Bonjour service discovery
2. Add optional access token protection
3. Add better range IO using seekable file descriptors where providers support it
4. Add media library indexing cache for large folders
5. Decide whether the Android app should reuse the desktop API shapes more strictly
