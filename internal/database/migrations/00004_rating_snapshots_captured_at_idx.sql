-- +goose Up
-- +goose StatementBegin
CREATE INDEX rating_snapshots_player_type_captured_idx
    ON rating_snapshots (player_id, rating_type, captured_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX rating_snapshots_player_type_captured_idx;
-- +goose StatementEnd
