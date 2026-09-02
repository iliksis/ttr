package server_test

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/iliksis/ttr/internal/server"
)

func getHTML(t *testing.T, handler http.Handler, path string) (*httptest.ResponseRecorder, string) {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, path, nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec, rec.Body.String()
}

// namePositions returns the index at which each name first appears in body,
// in the same order as names, failing the test if any name is missing.
func namePositions(t *testing.T, body string, names ...string) []int {
	t.Helper()

	positions := make([]int, len(names))
	for i, name := range names {
		pos := strings.Index(body, name)
		if pos == -1 {
			t.Fatalf("body does not contain %q, body = %s", name, body)
		}
		positions[i] = pos
	}
	return positions
}

func assertIncreasing(t *testing.T, positions []int, names []string) {
	t.Helper()

	for i := 1; i < len(positions); i++ {
		if positions[i-1] >= positions[i] {
			t.Fatalf("expected %q before %q, got positions %v for names %v", names[i-1], names[i], positions, names)
		}
	}
}

func TestHome_RendersValuesAndDatesAndDashForMissing(t *testing.T) {
	db := testDB(t)
	withHistory := seedManualPlayer(t, db, "Has", "History")
	seedSnapshot(t, db, withHistory, "TTR", 1550, "2026-01-02T00:00:00.000Z")
	seedSnapshot(t, db, withHistory, "QTTR", 1400, "2026-01-01T00:00:00.000Z")
	seedManualPlayer(t, db, "No", "History")
	handler := server.New(db, testIngestionKey)

	rec, body := getHTML(t, handler, "/")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, body)
	}
	if !strings.Contains(body, "1550") || !strings.Contains(body, "2026-01-02") {
		t.Fatalf("body missing TTR value/date, body = %s", body)
	}
	if !strings.Contains(body, "1400") || !strings.Contains(body, "2026-01-01") {
		t.Fatalf("body missing QTTR value/date, body = %s", body)
	}
	if !strings.Contains(body, "No") || !strings.Contains(body, "History") {
		t.Fatalf("body missing player with no history, body = %s", body)
	}
	if !strings.Contains(body, "-") {
		t.Fatalf("body missing dash placeholder for missing values, body = %s", body)
	}
}

func TestHome_DefaultSortIsQTTRDescending(t *testing.T) {
	db := testDB(t)
	low := seedManualPlayer(t, db, "Low", "Qttr")
	high := seedManualPlayer(t, db, "High", "Qttr")
	mid := seedManualPlayer(t, db, "Mid", "Qttr")
	seedSnapshot(t, db, low, "QTTR", 1000, "2026-01-01T00:00:00.000Z")
	seedSnapshot(t, db, high, "QTTR", 2000, "2026-01-01T00:00:00.000Z")
	seedSnapshot(t, db, mid, "QTTR", 1500, "2026-01-01T00:00:00.000Z")
	handler := server.New(db, testIngestionKey)

	_, body := getHTML(t, handler, "/")

	names := []string{"High Qttr", "Mid Qttr", "Low Qttr"}
	assertIncreasing(t, namePositions(t, body, names...), names)
}

func TestHome_SortByNameAscending(t *testing.T) {
	db := testDB(t)
	seedManualPlayer(t, db, "Charlie", "Zzz")
	seedManualPlayer(t, db, "Alice", "Zzz")
	seedManualPlayer(t, db, "Bob", "Zzz")
	handler := server.New(db, testIngestionKey)

	_, body := getHTML(t, handler, "/?sort=name&dir=asc")

	names := []string{"Alice Zzz", "Bob Zzz", "Charlie Zzz"}
	assertIncreasing(t, namePositions(t, body, names...), names)
}

func TestHome_SortByTTRDescending(t *testing.T) {
	db := testDB(t)
	low := seedManualPlayer(t, db, "Low", "Ttr")
	high := seedManualPlayer(t, db, "High", "Ttr")
	seedSnapshot(t, db, low, "TTR", 1000, "2026-01-01T00:00:00.000Z")
	seedSnapshot(t, db, high, "TTR", 2000, "2026-01-01T00:00:00.000Z")
	handler := server.New(db, testIngestionKey)

	_, body := getHTML(t, handler, "/?sort=ttr&dir=desc")

	names := []string{"High Ttr", "Low Ttr"}
	assertIncreasing(t, namePositions(t, body, names...), names)
}

func TestHome_SortByQTTRAscending(t *testing.T) {
	db := testDB(t)
	low := seedManualPlayer(t, db, "Low", "Qttr")
	high := seedManualPlayer(t, db, "High", "Qttr")
	seedSnapshot(t, db, low, "QTTR", 1000, "2026-01-01T00:00:00.000Z")
	seedSnapshot(t, db, high, "QTTR", 2000, "2026-01-01T00:00:00.000Z")
	handler := server.New(db, testIngestionKey)

	_, body := getHTML(t, handler, "/?sort=qttr&dir=asc")

	names := []string{"Low Qttr", "High Qttr"}
	assertIncreasing(t, namePositions(t, body, names...), names)
}

func TestHome_SortByTTRAscending(t *testing.T) {
	db := testDB(t)
	low := seedManualPlayer(t, db, "Low", "Ttr")
	high := seedManualPlayer(t, db, "High", "Ttr")
	seedSnapshot(t, db, low, "TTR", 1000, "2026-01-01T00:00:00.000Z")
	seedSnapshot(t, db, high, "TTR", 2000, "2026-01-01T00:00:00.000Z")
	handler := server.New(db, testIngestionKey)

	_, body := getHTML(t, handler, "/?sort=ttr&dir=asc")

	names := []string{"Low Ttr", "High Ttr"}
	assertIncreasing(t, namePositions(t, body, names...), names)
}

func TestHome_PlayersWithNoQTTRSortLastRegardlessOfDirection(t *testing.T) {
	db := testDB(t)
	withQTTR := seedManualPlayer(t, db, "With", "Qttr")
	seedManualPlayer(t, db, "Without", "Qttr")
	seedSnapshot(t, db, withQTTR, "QTTR", 1500, "2026-01-01T00:00:00.000Z")
	handler := server.New(db, testIngestionKey)

	for _, dir := range []string{"asc", "desc"} {
		_, body := getHTML(t, handler, "/?sort=qttr&dir="+dir)
		names := []string{"With Qttr", "Without Qttr"}
		assertIncreasing(t, namePositions(t, body, names...), names)
	}
}

func TestHome_SortByNameWithoutExplicitDirDefaultsAscending(t *testing.T) {
	db := testDB(t)
	seedManualPlayer(t, db, "Charlie", "Zzz")
	seedManualPlayer(t, db, "Alice", "Zzz")
	handler := server.New(db, testIngestionKey)

	_, body := getHTML(t, handler, "/?sort=name")

	names := []string{"Alice Zzz", "Charlie Zzz"}
	assertIncreasing(t, namePositions(t, body, names...), names)
}

func TestHome_LinksSharedStylesheet(t *testing.T) {
	db := testDB(t)
	handler := server.New(db, testIngestionKey)

	_, body := getHTML(t, handler, "/")

	if !strings.Contains(body, `href="/static/styles.css"`) {
		t.Fatalf("body missing stylesheet link, body = %s", body)
	}
}

func TestStaticStylesheet_IsServed(t *testing.T) {
	db := testDB(t)
	handler := server.New(db, testIngestionKey)

	rec, _ := getHTML(t, handler, "/static/styles.css")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if !strings.Contains(rec.Body.String(), "--color-primary") {
		t.Fatalf("stylesheet body missing expected token, body = %s", rec.Body.String())
	}
}

func TestHome_PlayerNameLinksToPlayerPage(t *testing.T) {
	db := testDB(t)
	id := seedManualPlayer(t, db, "Linked", "Player")
	handler := server.New(db, testIngestionKey)

	_, body := getHTML(t, handler, "/")

	want := "/players/" + strconv.FormatInt(id, 10)
	if !strings.Contains(body, want) {
		t.Fatalf("body missing link %q, body = %s", want, body)
	}
}
