package clubroster_test

import (
	"context"
	"path/filepath"
	"sync"
	"testing"

	"github.com/iliksis/ttr/internal/clubroster"
	"github.com/iliksis/ttr/internal/database"
	"github.com/iliksis/ttr/internal/mytt"
	"github.com/jmoiron/sqlx"
)

// fakeClient's FetchTeamPlayers is called concurrently by Fetch, so its call
// counters need a mutex.
type fakeClient struct {
	teams       []mytt.Team
	teamPlayers map[string][]mytt.Player

	mu           sync.Mutex
	teamsCalls   int
	playersCalls int
}

func (f *fakeClient) FetchTeams(ctx context.Context, clubNumber, organization string) ([]mytt.Team, error) {
	f.mu.Lock()
	f.teamsCalls++
	f.mu.Unlock()
	return f.teams, nil
}

func (f *fakeClient) FetchTeamPlayers(ctx context.Context, teamID string) ([]mytt.Player, error) {
	f.mu.Lock()
	f.playersCalls++
	f.mu.Unlock()
	return f.teamPlayers[teamID], nil
}

func testDB(t *testing.T) *sqlx.DB {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "ttr.db")
	db, err := database.Open(dbPath, database.Migrations)
	if err != nil {
		t.Fatalf("Open() error = %v, want nil", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func playerRow(t *testing.T, db *sqlx.DB, internalID string) (firstName, lastName string, found bool) {
	t.Helper()

	var rows []struct {
		FirstName string `db:"first_name"`
		LastName  string `db:"last_name"`
	}
	if err := db.Select(&rows, `SELECT first_name, last_name FROM players WHERE internal_id = ?`, internalID); err != nil {
		t.Fatalf("query player %s: %v", internalID, err)
	}
	if len(rows) == 0 {
		return "", "", false
	}
	return rows[0].FirstName, rows[0].LastName, true
}

func TestFetch_CreatesPlayersFromEveryTeam(t *testing.T) {
	db := testDB(t)
	client := &fakeClient{
		teams: []mytt.Team{{TeamID: "team-1"}, {TeamID: "team-2"}},
		teamPlayers: map[string][]mytt.Player{
			"team-1": {{InternalID: "NU1", FirstName: "Ada", LastName: "Lovelace"}},
			"team-2": {{InternalID: "NU2", FirstName: "Alan", LastName: "Turing"}},
		},
	}

	if err := clubroster.Fetch(context.Background(), db, client, "13118", "WTTV"); err != nil {
		t.Fatalf("Fetch() error = %v, want nil", err)
	}

	for id, wantFirst := range map[string]string{"NU1": "Ada", "NU2": "Alan"} {
		first, _, found := playerRow(t, db, id)
		if !found {
			t.Fatalf("player %s not created", id)
		}
		if first != wantFirst {
			t.Fatalf("player %s first_name = %s, want %s", id, first, wantFirst)
		}
	}

	var nuidCount int
	if err := db.Get(&nuidCount, `SELECT COUNT(*) FROM players WHERE internal_id = 'NU1' AND nuid IS NULL`); err != nil {
		t.Fatalf("nuid query error = %v, want nil", err)
	}
	if nuidCount != 1 {
		t.Fatalf("nuid for NU1 should stay unset, got %d matching rows", nuidCount)
	}
}

func TestFetch_IsIdempotent(t *testing.T) {
	db := testDB(t)
	client := &fakeClient{
		teams: []mytt.Team{{TeamID: "team-1"}},
		teamPlayers: map[string][]mytt.Player{
			"team-1": {{InternalID: "NU1", FirstName: "Ada", LastName: "Lovelace"}},
		},
	}

	if err := clubroster.Fetch(context.Background(), db, client, "13118", "WTTV"); err != nil {
		t.Fatalf("Fetch() [1] error = %v, want nil", err)
	}
	if err := clubroster.Fetch(context.Background(), db, client, "13118", "WTTV"); err != nil {
		t.Fatalf("Fetch() [2] error = %v, want nil", err)
	}

	var count int
	if err := db.Get(&count, `SELECT COUNT(*) FROM players WHERE internal_id = 'NU1'`); err != nil {
		t.Fatalf("count query error = %v, want nil", err)
	}
	if count != 1 {
		t.Fatalf("player count = %d, want 1 after re-fetch", count)
	}
}

func TestFetch_SkipsPlayersWithEmptyInternalID(t *testing.T) {
	db := testDB(t)
	client := &fakeClient{
		teams: []mytt.Team{{TeamID: "team-1"}},
		teamPlayers: map[string][]mytt.Player{
			"team-1": {
				{InternalID: "", FirstName: "Nobody", LastName: "Placeholder"},
				{InternalID: "NU1", FirstName: "Ada", LastName: "Lovelace"},
			},
		},
	}

	if err := clubroster.Fetch(context.Background(), db, client, "13118", "WTTV"); err != nil {
		t.Fatalf("Fetch() error = %v, want nil", err)
	}

	var emptyCount int
	if err := db.Get(&emptyCount, `SELECT COUNT(*) FROM players WHERE internal_id = ''`); err != nil {
		t.Fatalf("count query error = %v, want nil", err)
	}
	if emptyCount != 0 {
		t.Fatalf("players with empty internal_id = %d, want 0", emptyCount)
	}

	first, _, found := playerRow(t, db, "NU1")
	if !found || first != "Ada" {
		t.Fatalf("player NU1 = (%s, found=%v), want Ada", first, found)
	}
}

func TestFetch_UpdatesNameOnRefetch(t *testing.T) {
	db := testDB(t)
	client := &fakeClient{
		teams: []mytt.Team{{TeamID: "team-1"}},
		teamPlayers: map[string][]mytt.Player{
			"team-1": {{InternalID: "NU1", FirstName: "Ada", LastName: "Lovelace"}},
		},
	}
	if err := clubroster.Fetch(context.Background(), db, client, "13118", "WTTV"); err != nil {
		t.Fatalf("Fetch() [1] error = %v, want nil", err)
	}

	client.teamPlayers["team-1"] = []mytt.Player{{InternalID: "NU1", FirstName: "Augusta", LastName: "Lovelace"}}
	if err := clubroster.Fetch(context.Background(), db, client, "13118", "WTTV"); err != nil {
		t.Fatalf("Fetch() [2] error = %v, want nil", err)
	}

	first, _, found := playerRow(t, db, "NU1")
	if !found {
		t.Fatal("player NU1 not found after re-fetch")
	}
	if first != "Augusta" {
		t.Fatalf("first_name = %s, want Augusta (updated in place)", first)
	}
}

func TestFetch_PlayerAbsentFromNewRunKeepsRowAndHistory(t *testing.T) {
	db := testDB(t)
	client := &fakeClient{
		teams: []mytt.Team{{TeamID: "team-1"}},
		teamPlayers: map[string][]mytt.Player{
			"team-1": {
				{InternalID: "NU1", FirstName: "Ada", LastName: "Lovelace"},
				{InternalID: "NU2", FirstName: "Alan", LastName: "Turing"},
			},
		},
	}
	if err := clubroster.Fetch(context.Background(), db, client, "13118", "WTTV"); err != nil {
		t.Fatalf("Fetch() [1] error = %v, want nil", err)
	}

	var playerID int64
	if err := db.Get(&playerID, `SELECT id FROM players WHERE internal_id = 'NU2'`); err != nil {
		t.Fatalf("lookup NU2 id: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO rating_snapshots (player_id, rating_type, value) VALUES (?, 'TTR', 1500)`, playerID); err != nil {
		t.Fatalf("seed snapshot: %v", err)
	}

	// NU2 no longer appears on the team.
	client.teamPlayers["team-1"] = []mytt.Player{{InternalID: "NU1", FirstName: "Ada", LastName: "Lovelace"}}
	if err := clubroster.Fetch(context.Background(), db, client, "13118", "WTTV"); err != nil {
		t.Fatalf("Fetch() [2] error = %v, want nil", err)
	}

	_, _, found := playerRow(t, db, "NU2")
	if !found {
		t.Fatal("player NU2 was removed, want row kept when absent from a fresh fetch")
	}

	var snapshotCount int
	if err := db.Get(&snapshotCount, `SELECT COUNT(*) FROM rating_snapshots WHERE player_id = ?`, playerID); err != nil {
		t.Fatalf("snapshot count query error = %v, want nil", err)
	}
	if snapshotCount != 1 {
		t.Fatalf("snapshot count = %d, want 1 (history must be untouched)", snapshotCount)
	}
}
