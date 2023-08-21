
CREATE TABLE refresh_tokens (
    id UUID NOT NULL DEFAULT uuid_generate_v4(),
    private_key_id UUID NOT NULL,
    plaintext text NOT NULL,

    created_at TIMESTAMPTZ(6) NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ(6) NOT NULL,

    PRIMARY KEY(id),
    FOREIGN KEY(private_key_id) REFERENCES private_keys(id),
    UNIQUE(private_key_id)
);

CREATE TABLE user_refresh_tokens (
    user_id UUID NOT NULL,
    refresh_token_id UUID NOT NULL,

    FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY(refresh_token_id) REFERENCES refresh_tokens(id) ON DELETE CASCADE,
    UNIQUE(user_id, refresh_token_id),
    UNIQUE(refresh_token_id)
);
