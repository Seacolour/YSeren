# YSeren Android

This module is YSeren's Android media-source application. It keeps media on the
device and exposes only supported files to browsers on the local network.

## Goal

Let an Android device expose user-selected local media over HTTP to the local
network, so browsers or dedicated media players can consume the stream directly.

## App Experience

The second-generation Compose UI follows the same product structure as the
desktop app, adapted to bottom navigation:

- **Share**: service state, selected-source summary, local browser preview,
  copyable LAN addresses, and start/stop controls.
- **Sources**: select, replace, remove, and rescan one SAF directory; the UI
  shows a readable storage path instead of the underlying `content://` URI.
- **Settings**: edit the port, automatically restart an active share, and view
  foreground-service and battery guidance.

The app persists read permission for the selected directory and runs a
lightweight HTTP server in an Android foreground service. The embedded Svelte
Web Player remains responsible for browsing and playback.

## HTTP Contract

- `/` packaged YSeren Web Player, with a diagnostic landing page fallback
- `/api/status` sanitized current server information
- `/api/version` app version and release URL
- `/api/tree` recursive cross-platform media tree
- `/api/tree?q=...` filtered media tree
- `/api/tree?path=...` compatibility/diagnostic directory listing
- `/playlist.m3u` recursive media playlist
- `/stream/android/<relative-path>` canonical byte-range media URL
- `/stream/<relative-path>` legacy Android URL retained for compatibility

Tree timestamps use Unix seconds. Streaming supports `GET`, `HEAD`, bounded,
open-ended, and suffix byte ranges; malformed or unsatisfiable ranges return
`416` with `Content-Range: bytes */<total>`.

## Important Limitations

- This MVP depends on Android's Storage Access Framework, so the app only sees
  files and folders the user explicitly grants access to.
- The share service is designed to run in the foreground with a notification.
- Range support uses SAF stream skipping and is not yet optimized for very large
  seek-heavy files or providers with slow random access.
- No transcoding is included. The app focuses only on exposing the original
  media source to the LAN.
- The current product supports one Android media source, represented by the
  stable contract source name `android`.
- There is no access token yet. Only run the share on a trusted local network.

## Build

Open the `android/` folder in Android Studio. The Gradle build runs with JDK 17
or 21 and compiles the app for Java 17.

To package the Web UI into the Android APK, build the frontend first:

```powershell
cd ..\frontend
npm install
npm run build
```

The Android Gradle project includes `../frontend/dist` as app assets. If that
directory is missing when the APK is built, `/` falls back to the diagnostic
landing page.

From PowerShell in `android/`:

```powershell
.\gradlew.bat :app:testDebugUnitTest :app:assembleDebug :app:assembleRelease --console=plain --no-daemon
```

Local builds use `versionName=dev`. A formal release injects the shared tag
version and a monotonically increasing Android version code:

```powershell
.\gradlew.bat :app:assembleRelease `
  '-PyserenVersionName=0.2.0' `
  '-PyserenVersionCode=2000' `
  --no-daemon
```

Without release signing variables, the local release output is
`app/build/outputs/apk/release/app-release-unsigned.apk`.

## Validation Status

As of 2026-07-24, the Android app has been exercised on both a LeiDian Android 9
emulator and a physical Android device. The emulator SAF directory contained
three MP4 files and one ZIP archive; the three media files were indexed and the
archive was ignored. Debug unit tests, debug assembly, release lint, and
unsigned release assembly pass.

HTTP checks cover full `GET`, `HEAD`, bounded/open-ended/suffix ranges, `416`,
legacy stream URLs, and method rejection. The Windows browser reached the
Android service through ADB port forwarding and played an MP4 with the embedded
Web Player. The app also opened the system browser successfully.

The emulator uses NAT, so its reported LAN address could not be reached directly
from Windows. The remaining real-device gate was subsequently completed: a
physical Android device shared 24 media files from `Internal storage / Movies`,
and Windows Chrome reached the Web Player at `http://192.168.50.171:1479/`,
browsed the Android source, and played media successfully over Wi-Fi. Phase 3 is
therefore complete.

## Release Signing

Release builds can be signed by providing these environment variables:

- `ANDROID_KEYSTORE_PATH`
- `ANDROID_KEYSTORE_PASSWORD`
- `ANDROID_KEY_ALIAS`
- `ANDROID_KEY_PASSWORD`

The GitHub Release workflow expects the matching repository secrets:

- `ANDROID_KEYSTORE_BASE64`
- `ANDROID_KEYSTORE_PASSWORD`
- `ANDROID_KEY_ALIAS`
- `ANDROID_KEY_PASSWORD`

If signing secrets are not configured, the workflow still builds and uploads an
unsigned APK for testing or manual side loading.
