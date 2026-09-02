// Content script entry point. Pings the background service worker on every
// mytischtennis.de page load; on greenlight, runs the sequential per-Player
// Capture and reports the batch back. Kept thin — behavior lives in
// content-core.js.
import browser from "webextension-polyfill";
import { captureRoster, fetchHistory, AuthError } from "./content-core.js";

async function run() {
  const pingResponse = await browser.runtime.sendMessage({ type: "PING" });
  if (!pingResponse?.ran) {
    return;
  }

  try {
    const results = await captureRoster(
      {
        fetchHistory: (nuid) => fetchHistory(fetch, nuid),
        sleep: (ms) => new Promise((resolve) => setTimeout(resolve, ms)),
      },
      pingResponse.nuids,
    );
    await browser.runtime.sendMessage({ type: "BATCH", results });
  } catch (err) {
    if (err instanceof AuthError) {
      console.warn("[ttr] capture aborted:", err.message);
      return;
    }
    throw err;
  }
}

run();
