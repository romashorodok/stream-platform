
CREATE TABLE private_keys (
    id UUID NOT NULL DEFAULT uuid_generate_v4(),
    jws_message json NOT NULL,

    PRIMARY KEY(id)
);

CREATE TABLE user_private_keys (
    user_id UUID NOT NULL,
    private_key_id UUID NOT NULL,

    FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY(private_key_id) REFERENCES private_keys(id) ON DELETE CASCADE,
    UNIQUE(user_id, private_key_id),
    UNIQUE(private_key_id)
);

