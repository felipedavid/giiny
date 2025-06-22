
CREATE TABLE bot_config (
    id TEXT PRIMARY KEY,
    username TEXT NOT NULL DEFAULT '',
    plain_password TEXT NOT NULL DEFAULT '',
    ai_prompt TEXT NOT NULL DEFAULT '',
    ai_api_key TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

INSERT INTO bot_config (id) VALUES ('11111111-1111-1111-1111-111111111111');