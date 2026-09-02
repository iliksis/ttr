-- +goose Up
-- +goose StatementBegin
CREATE UNIQUE INDEX players_internal_id_idx ON players (internal_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX players_internal_id_idx;
-- +goose StatementEnd
