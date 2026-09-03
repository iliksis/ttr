import { describe, expect, it, vi } from "vitest";
import {
  HISTORY_SPACING_MS,
  captureRoster,
  isAuthFailure,
} from "../src/content-core.js";

function historyOf(ttr, vq_ttr) {
  return { ttr, vq_ttr };
}

describe("isAuthFailure", () => {
  it("recognizes mytischtennis.de's 200-with-error-body auth failure shape", () => {
    expect(isAuthFailure({ ttr: null, error: { type: "PT403", message: "Not authorized" } })).toBe(true);
  });

  it("treats a normal history payload as not an auth failure", () => {
    expect(isAuthFailure(historyOf(1500, 1510))).toBe(false);
  });
});

describe("captureRoster", () => {
  it("fetches history for every nuid sequentially and batches the results", async () => {
    const fetchHistory = vi.fn(async (nuid) =>
      nuid === "NU1" ? historyOf(1500, 1510) : historyOf(1600, 1610),
    );
    const sleep = vi.fn(async () => {});

    const { results, authFailed } = await captureRoster({ fetchHistory, sleep }, ["NU1", "NU2"]);

    expect(results).toEqual([
      { nuid: "NU1", ttr: 1500, qttr: 1510 },
      { nuid: "NU2", ttr: 1600, qttr: 1610 },
    ]);
    expect(authFailed).toBe(false);
    expect(fetchHistory).toHaveBeenCalledTimes(2);
  });

  it("spaces calls ~250ms apart, but not before the first call", async () => {
    const fetchHistory = vi.fn(async () => historyOf(1500, 1510));
    const sleep = vi.fn(async () => {});

    await captureRoster({ fetchHistory, sleep }, ["NU1", "NU2", "NU3"]);

    expect(sleep).toHaveBeenCalledTimes(2);
    expect(sleep).toHaveBeenCalledWith(HISTORY_SPACING_MS);
  });

  it("sends a single batched result set, not one call per Player", async () => {
    const fetchHistory = vi.fn(async () => historyOf(1500, 1510));
    const sleep = vi.fn(async () => {});
    const nuids = ["NU1", "NU2", "NU3"];

    const { results } = await captureRoster({ fetchHistory, sleep }, nuids);

    expect(results).toHaveLength(3);
  });

  it("stops on an auth failure without misreporting success, but keeps Ratings already captured", async () => {
    const fetchHistory = vi.fn(async (nuid) =>
      nuid === "NU2"
        ? { ttr: null, error: { type: "PT403", message: "Not authorized" } }
        : historyOf(1500, 1510),
    );
    const sleep = vi.fn(async () => {});

    const { results, authFailed } = await captureRoster(
      { fetchHistory, sleep },
      ["NU1", "NU2", "NU3"],
    );

    expect(authFailed).toBe(true);
    expect(results).toEqual([{ nuid: "NU1", ttr: 1500, qttr: 1510 }]);
    expect(fetchHistory).toHaveBeenCalledTimes(2);
  });

  it("skips a Player whose response is missing a Rating field, without aborting the run", async () => {
    const fetchHistory = vi.fn(async (nuid) =>
      nuid === "NU2" ? { ttr: 1500 } : historyOf(1500, 1510),
    );
    const sleep = vi.fn(async () => {});

    const { results } = await captureRoster({ fetchHistory, sleep }, ["NU1", "NU2", "NU3"]);

    expect(results.map((r) => r.nuid)).toEqual(["NU1", "NU3"]);
  });

  it("returns an empty batch for an empty roster without sleeping", async () => {
    const fetchHistory = vi.fn();
    const sleep = vi.fn();

    const { results, authFailed } = await captureRoster({ fetchHistory, sleep }, []);

    expect(results).toEqual([]);
    expect(authFailed).toBe(false);
    expect(fetchHistory).not.toHaveBeenCalled();
    expect(sleep).not.toHaveBeenCalled();
  });
});
