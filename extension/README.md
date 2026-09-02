# TTR Extension

Captures TTR/QTTR Rating snapshots while browsing mytischtennis.de and reports
them to the TTR Server. Single MV3 codebase for Chrome and Firefox, loaded
unpacked — not published to either store.

## Develop

```
npm install
npm test
```

`src/background-core.js` and `src/content-core.js` hold the testable logic
(pure functions with injected `fetch`/storage/timers); `background.js` and
`content.js` just wire that logic to the real WebExtension APIs.

## Load unpacked

```
npm run build
```

builds `dist/` — load that folder as an unpacked extension:

- **Chrome**: `chrome://extensions` → enable Developer mode → "Load unpacked" → select `extension/dist`.
- **Firefox**: `about:debugging#/runtime/this-firefox` → "Load Temporary Add-on" → select `extension/dist/manifest.json`.

The Server URL and Ingestion key are hardcoded dev-config values in
`src/config.js` for this build; a real options page and persisted key land in
issue #17.
