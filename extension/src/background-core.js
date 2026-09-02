// Testable core of the background service worker: the once-per-day gate,
// the roster fetch, and the Ingestion POST. Every browser/network dependency
// is injected so this can be unit-tested with a mocked fetch and a mocked
// storage object, per the spec's testing seam (no real browser).

export const LAST_CAPTURE_DATE_KEY = "ttr:lastCaptureDate";

// todayString reduces a Date to a local calendar-day key so the once-per-day
// gate compares days, not instants.
export function todayString(date) {
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, "0");
  const day = String(date.getDate()).padStart(2, "0");
  return `${year}-${month}-${day}`;
}

// handlePing decides whether today's Capture run should proceed. On
// greenlight it records today as the last-run day before fetching the
// roster, so a run that starts today never runs again today even if the
// rest of the Capture fails downstream (retry/backoff is #17's job).
//
// deps.storage: { get(key): Promise<string|undefined>, set(key, value): Promise<void> }
// deps.fetchRoster: () => Promise<string[]>
// deps.now: () => Date
export async function handlePing(deps) {
  const today = todayString(deps.now());
  const lastRun = await deps.storage.get(LAST_CAPTURE_DATE_KEY);
  if (lastRun === today) {
    return { ran: false };
  }

  await deps.storage.set(LAST_CAPTURE_DATE_KEY, today);
  const nuids = await deps.fetchRoster();
  return { ran: true, nuids };
}

// fetchRoster fetches the current roster fresh from the Server's
// GET /api/roster — no local caching — and returns just the nuid list.
export async function fetchRoster(fetchImpl, serverURL) {
  const res = await fetchImpl(`${serverURL}/api/roster`);
  if (!res.ok) {
    throw new Error(`roster fetch failed: ${res.status}`);
  }
  const roster = await res.json();
  return roster.map((entry) => entry.nuid);
}

// postIngestBatch POSTs a Capture batch to the Server's Ingestion endpoint.
export async function postIngestBatch(fetchImpl, serverURL, ingestionKey, results) {
  const body = results.map(({ nuid, ttr, qttr }) => ({ nuid, ttr, qttr }));
  const res = await fetchImpl(`${serverURL}/api/ingest`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${ingestionKey}`,
    },
    body: JSON.stringify(body),
  });
  if (!res.ok) {
    throw new Error(`ingest post failed: ${res.status}`);
  }
  return res.json();
}
