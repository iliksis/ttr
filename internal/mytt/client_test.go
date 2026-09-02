package mytt_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/iliksis/ttr/internal/mytt"
)

func TestFetchTeams_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/ttr/teams" {
			t.Fatalf("path = %s, want /api/ttr/teams", r.URL.Path)
		}
		if got := r.URL.Query().Get("clubNumber"); got != "13118" {
			t.Fatalf("clubNumber = %s, want 13118", got)
		}
		if got := r.URL.Query().Get("organization"); got != "WTTV" {
			t.Fatalf("organization = %s, want WTTV", got)
		}
		fmt.Fprint(w, `{"data":[{"team_id":"2953148"},{"team_id":"2953149"}],"error":null}`)
	}))
	defer srv.Close()

	client := mytt.NewClient(srv.URL)
	teams, err := client.FetchTeams(context.Background(), "13118", "WTTV")
	if err != nil {
		t.Fatalf("FetchTeams() error = %v, want nil", err)
	}

	if len(teams) != 2 {
		t.Fatalf("len(teams) = %d, want 2", len(teams))
	}
	if teams[0].TeamID != "2953148" {
		t.Fatalf("teams[0].TeamID = %s, want 2953148", teams[0].TeamID)
	}
}

func TestFetchTeamPlayers_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/ttr/team/players" {
			t.Fatalf("path = %s, want /api/ttr/team/players", r.URL.Path)
		}
		if got := r.URL.Query().Get("teamId"); got != "2953148" {
			t.Fatalf("teamId = %s, want 2953148", got)
		}
		fmt.Fprint(w, `{"data":[{"firstname":"Dang","lastname":"Qiu","internal_id":"NU7535","rank":1}],"error":null}`)
	}))
	defer srv.Close()

	client := mytt.NewClient(srv.URL)
	players, err := client.FetchTeamPlayers(context.Background(), "2953148")
	if err != nil {
		t.Fatalf("FetchTeamPlayers() error = %v, want nil", err)
	}

	if len(players) != 1 {
		t.Fatalf("len(players) = %d, want 1", len(players))
	}
	want := mytt.Player{InternalID: "NU7535", FirstName: "Dang", LastName: "Qiu"}
	if players[0] != want {
		t.Fatalf("players[0] = %+v, want %+v", players[0], want)
	}
}

func TestFetchTeams_UpstreamErrorField(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"data":[],"error":"invalid club number"}`)
	}))
	defer srv.Close()

	client := mytt.NewClient(srv.URL)
	if _, err := client.FetchTeams(context.Background(), "bad", "WTTV"); err == nil {
		t.Fatal("FetchTeams() error = nil, want non-nil when upstream sets the error field")
	}
}

func TestFetchTeams_ErrorStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := mytt.NewClient(srv.URL)
	if _, err := client.FetchTeams(context.Background(), "13118", "WTTV"); err == nil {
		t.Fatal("FetchTeams() error = nil, want non-nil on upstream 500")
	}
}
