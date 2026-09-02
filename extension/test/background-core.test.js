import { describe, expect, it, vi } from "vitest";
import {
  LAST_CAPTURE_DATE_KEY,
  fetchRoster,
  handlePing,
  postIngestBatch,
  todayString,
} from "../src/background-core.js";

function fakeStorage(initial = {}) {
  const data = { ...initial };
  return {
    data,
    get: vi.fn(async (key) => data[key]),
    set: vi.fn(async (key, value) => {
      data[key] = value;
    }),
  };
}

describe("todayString", () => {
  it("formats a Date as a local calendar-day key", () => {
    expect(todayString(new Date(2026, 8, 2, 23, 59))).toBe("2026-09-02");
  });
});

describe("handlePing", () => {
  it("no-ops when Capture already ran today", async () => {
    const storage = fakeStorage({ [LAST_CAPTURE_DATE_KEY]: "2026-09-02" });
    const fetchRoster = vi.fn();

    const result = await handlePing({
      storage,
      fetchRoster,
      now: () => new Date(2026, 8, 2, 8, 0),
    });

    expect(result).toEqual({ ran: false });
    expect(fetchRoster).not.toHaveBeenCalled();
  });

  it("greenlights, records today, and fetches the roster when not yet run today", async () => {
    const storage = fakeStorage({ [LAST_CAPTURE_DATE_KEY]: "2026-09-01" });
    const fetchRoster = vi.fn(async () => ["NU1", "NU2"]);

    const result = await handlePing({
      storage,
      fetchRoster,
      now: () => new Date(2026, 8, 2, 8, 0),
    });

    expect(result).toEqual({ ran: true, nuids: ["NU1", "NU2"] });
    expect(storage.set).toHaveBeenCalledWith(LAST_CAPTURE_DATE_KEY, "2026-09-02");
  });

  it("greenlights on first-ever run with no stored last-run date", async () => {
    const storage = fakeStorage();
    const fetchRoster = vi.fn(async () => []);

    const result = await handlePing({
      storage,
      fetchRoster,
      now: () => new Date(2026, 8, 2, 8, 0),
    });

    expect(result.ran).toBe(true);
  });

  it("records today before fetching the roster, so the gate holds even if the fetch fails", async () => {
    const storage = fakeStorage();
    const fetchRoster = vi.fn(async () => {
      throw new Error("network down");
    });

    await expect(
      handlePing({ storage, fetchRoster, now: () => new Date(2026, 8, 2) }),
    ).rejects.toThrow("network down");

    expect(storage.data[LAST_CAPTURE_DATE_KEY]).toBe("2026-09-02");
  });
});

describe("fetchRoster", () => {
  it("fetches from the Server's /api/roster and returns just the nuid list", async () => {
    const fetchImpl = vi.fn(async (url) => {
      expect(url).toBe("http://server.example/api/roster");
      return {
        ok: true,
        json: async () => [{ nuid: "NU1" }, { nuid: "NU2" }],
      };
    });

    const nuids = await fetchRoster(fetchImpl, "http://server.example");

    expect(nuids).toEqual(["NU1", "NU2"]);
  });

  it("throws on a non-OK response", async () => {
    const fetchImpl = vi.fn(async () => ({ ok: false, status: 500 }));

    await expect(fetchRoster(fetchImpl, "http://server.example")).rejects.toThrow(
      "roster fetch failed: 500",
    );
  });
});

describe("postIngestBatch", () => {
  it("POSTs the batch to /api/ingest with the Ingestion key and stripped-down entries", async () => {
    const fetchImpl = vi.fn(async (url, init) => {
      expect(url).toBe("http://server.example/api/ingest");
      expect(init.method).toBe("POST");
      expect(init.headers.Authorization).toBe("Bearer secret-key");
      expect(JSON.parse(init.body)).toEqual([
        { nuid: "NU1", ttr: 1500, qttr: 1510 },
      ]);
      return { ok: true, json: async () => ({ results: [] }) };
    });

    await postIngestBatch(fetchImpl, "http://server.example", "secret-key", [
      { nuid: "NU1", ttr: 1500, qttr: 1510, extra: "ignored" },
    ]);

    expect(fetchImpl).toHaveBeenCalledTimes(1);
  });

  it("throws on a non-OK response", async () => {
    const fetchImpl = vi.fn(async () => ({ ok: false, status: 401 }));

    await expect(
      postIngestBatch(fetchImpl, "http://server.example", "key", []),
    ).rejects.toThrow("ingest post failed: 401");
  });
});
