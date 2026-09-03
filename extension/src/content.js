// Content script entry point. Pings the background service worker on every
// mytischtennis.de page load; on greenlight, runs the sequential per-Player
// Capture and reports the batch back. Kept thin — behavior lives in
// content-core.js.
import browser from "webextension-polyfill";
import { captureRoster, fetchHistory } from "./content-core.js";

async function run() {
  const pingResponse = await browser.runtime.sendMessage({ type: "PING" });
  if (!pingResponse?.ran) {
    return;
  }

  const { results, authFailed } = await captureRoster(
    {
      fetchHistory: (nuid) => fetchHistory(fetch, nuid),
      sleep: (ms) => new Promise((resolve) => setTimeout(resolve, ms)),
    },
    pingResponse.nuids,
  );

  if (authFailed) {
    console.warn("[ttr] capture stopped early: auth failure fetching history");
  }
  if (results.length > 0) {
    const batchResponse = await browser.runtime.sendMessage({ type: "BATCH", results });
    if (!batchResponse?.ok) {
      console.warn("[ttr] capture batch failed to send:", batchResponse?.error);
    }
  }
}

run();
