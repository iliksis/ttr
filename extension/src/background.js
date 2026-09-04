// Background service worker entry point. Wires background-core's testable
// logic to the real browser storage/messaging APIs; kept intentionally thin
// so behavior lives in background-core.js instead.
import browser from "webextension-polyfill";
import { SERVER_URL, INGESTION_KEY } from "./config.js";
import { handlePing, fetchRoster, postIngestBatch } from "./background-core.js";

const storage = {
  async get(key) {
    const stored = await browser.storage.local.get(key);
    return stored[key];
  },
  async set(key, value) {
    await browser.storage.local.set({ [key]: value });
  },
};

browser.runtime.onMessage.addListener((message) => {
  if (message?.type === "PING") {
    return handlePing({
      storage,
      fetchRoster: () => fetchRoster(fetch, SERVER_URL),
      now: () => new Date(),
    });
  }

  if (message?.type === "BATCH") {
    return postIngestBatch(fetch, SERVER_URL, INGESTION_KEY, message.results)
      .then((response) => ({ ok: true, results: response.results }))
      .catch((err) => ({ ok: false, error: String(err) }));
  }

  return undefined;
});
