# Android Share MVP

This module is the Android-side prototype for YSeren as a mobile media source.

## Goal

Let an Android device expose user-selected local media over HTTP to the local
network, so browsers or dedicated media players can consume the stream directly.

## MVP Scope

- User selects a folder through the Android system file picker.
- The app stores persistent read permission for that folder.
- A foreground service starts a lightweight local HTTP server.
- The server exposes:
  - `/` packaged YSeren Web UI, with a simple diagnostic landing page fallback
  - `/api/status` current server info
  - `/api/tree` recursive media tree for the Web UI
  - `/api/tree?path=...` directory listing for diagnostics
  - `/playlist.m3u` recursive media playlist
  - `/stream/<relative-path>` byte-range-friendly media streaming

## Important Limitations

- This MVP depends on Android's Storage Access Framework, so the app only sees
  files and folders the user explicitly grants access to.
- The share service is designed to run in the foreground with a notification.
- Range support is implemented with stream skipping, which is acceptable for an
  MVP but not yet optimized for very large seek-heavy files.
- No transcoding is included. The app focuses only on exposing the original
  media source to the LAN.

## Build

Open the `android/` folder in Android Studio.

To package the Web UI into the Android APK, build the frontend first:

```powershell
cd ..\frontend
npm install
npm run build
```

The Android Gradle project includes `../frontend/dist` as app assets. If that
directory is missing when the APK is built, `/` falls back to the diagnostic
landing page.

The repository currently includes the Android project files, but build
verification still depends on a local Android SDK installation.
