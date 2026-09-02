-- +goose Up
-- +goose StatementBegin
CREATE TABLE players (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    nuid TEXT UNIQUE,
    internal_id TEXT,
    first_name TEXT NOT NULL,
    last_name TEXT NOT NULL
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE rating_snapshots (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    player_id INTEGER NOT NULL REFERENCES players(id),
    rating_type TEXT NOT NULL CHECK (rating_type IN ('TTR', 'QTTR')),
    value INTEGER NOT NULL,
    captured_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE UNIQUE INDEX rating_snapshots_player_type_day_idx
    ON rating_snapshots (player_id, rating_type, date(captured_at));
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE rating_snapshots;
-- +goose StatementEnd

-- +goose StatementBegin
DROP TABLE players;
-- +goose StatementEnd
