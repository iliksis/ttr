// Testable core of the content script: sequential per-Player history
// fetches, spaced ~250ms apart, batched into one result set. All I/O is
// injected so this can be unit-tested with a mocked fetch, per the spec's
// testing seam (no real page DOM, no browser test runner).

export const HISTORY_SPACING_MS = 250;

// AuthError signals mytischtennis.de rejected the request for the current
// session cookie. mytischtennis.de reports this as HTTP 200 with an error
// object in the body rather than a 401/403, so callers must check the body.
export class AuthError extends Error {}

// isAuthFailure reports whether a decoded history response is
// mytischtennis.de's 200-with-error-body auth failure shape, rather than a
// real history payload.
export function isAuthFailure(historyResponse) {
  return Boolean(historyResponse && historyResponse.error);
}

// hasRatings reports whether a history response carries both Rating fields
// Capture needs. A response missing either is skipped rather than reported,
// since sending a bogus value would misreport a Rating snapshot.
function hasRatings(historyResponse) {
  return (
    typeof historyResponse?.ttr === "number" &&
    typeof historyResponse?.vq_ttr === "number"
  );
}

// captureRoster fetches history for every nuid in the roster, sequentially,
// waiting HISTORY_SPACING_MS between calls (not before the first), and
// returns the batched results once every call has settled. It aborts
// immediately on an auth failure, since one bad session cookie invalidates
// every remaining call in the run.
//
// deps.fetchHistory: (nuid) => Promise<object>
// deps.sleep: (ms) => Promise<void>
export async function captureRoster(deps, nuids) {
  const results = [];
  for (let i = 0; i < nuids.length; i++) {
    if (i > 0) {
      await deps.sleep(HISTORY_SPACING_MS);
    }

    const nuid = nuids[i];
    const data = await deps.fetchHistory(nuid);

    if (isAuthFailure(data)) {
      throw new AuthError(`auth failure fetching history for ${nuid}`);
    }
    if (!hasRatings(data)) {
      continue;
    }

    results.push({ nuid, ttr: data.ttr, qttr: data.vq_ttr });
  }
  return results;
}

// fetchHistory calls mytischtennis.de's per-Player history endpoint, riding
// the page's own session cookie.
export async function fetchHistory(fetchImpl, nuid) {
  const res = await fetchImpl(
    `https://www.mytischtennis.de/api/ttr/history/${nuid}`,
    { credentials: "include" },
  );
  return res.json();
}
