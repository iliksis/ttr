// Package mytt is a client for mytischtennis.de's public, unauthenticated
// team/roster API.
package mytt

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// DefaultBaseURL is mytischtennis.de's production API host.
const DefaultBaseURL = "https://www.mytischtennis.de"

// Team is one entry from GET /api/ttr/teams. Only the fields the club
// roster sync needs are modeled; the upstream response carries more.
type Team struct {
	TeamID string `json:"team_id"`
}

// Player is one entry from GET /api/ttr/team/players.
type Player struct {
	InternalID string `json:"internal_id"`
	FirstName  string `json:"firstname"`
	LastName   string `json:"lastname"`
}

type response[T any] struct {
	Data  []T     `json:"data"`
	Error *string `json:"error"`
}

// requestTimeout bounds every upstream call so a stalled mytischtennis.de
// connection can't wedge the daily sync job forever.
const requestTimeout = 30 * time.Second

// Client fetches teams and team rosters from mytischtennis.de.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient builds a Client. An empty baseURL defaults to DefaultBaseURL;
// tests pass an httptest.Server URL instead.
func NewClient(baseURL string) *Client {
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	return &Client{baseURL: baseURL, httpClient: &http.Client{Timeout: requestTimeout}}
}

// FetchTeams retrieves every team registered under clubNumber within
// organization (e.g. "WTTV").
func (c *Client) FetchTeams(ctx context.Context, clubNumber, organization string) ([]Team, error) {
	q := url.Values{"clubNumber": {clubNumber}, "organization": {organization}}
	var out response[Team]
	if err := c.get(ctx, "/api/ttr/teams", q, &out); err != nil {
		return nil, fmt.Errorf("fetch teams: %w", err)
	}
	return out.Data, nil
}

// FetchTeamPlayers retrieves the roster for a single team.
func (c *Client) FetchTeamPlayers(ctx context.Context, teamID string) ([]Player, error) {
	q := url.Values{"teamId": {teamID}}
	var out response[Player]
	if err := c.get(ctx, "/api/ttr/team/players", q, &out); err != nil {
		return nil, fmt.Errorf("fetch team %s players: %w", teamID, err)
	}
	return out.Data, nil
}

// get performs the request and decodes its body into out, which must embed
// the {"data": ..., "error": ...} envelope every mytischtennis.de API
// response uses. An "error" field set in an otherwise-200 response is
// upstream reporting a failure (e.g. an invalid clubNumber), so it's
// surfaced as a Go error rather than treated as an empty result.
func (c *Client) get(ctx context.Context, path string, query url.Values, out interface {
	upstreamError() *string
}) error {
	reqURL := c.baseURL + path + "?" + query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}

	if errMsg := out.upstreamError(); errMsg != nil {
		return fmt.Errorf("upstream error: %s", *errMsg)
	}

	return nil
}

func (r *response[T]) upstreamError() *string { return r.Error }
