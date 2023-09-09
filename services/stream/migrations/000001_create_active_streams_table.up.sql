
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE active_streams (
    id UUID NOT NULL DEFAULT uuid_generate_v4(),

    running BOOLEAN NOT NULL DEFAULT FALSE,
    deployed BOOLEAN NOT NULL DEFAULT FALSE,

    broadcaster_id UUID NOT NULL UNIQUE,
    username VARCHAR(30) NOT NULL UNIQUE,

    namespace VARCHAR(50) NOT NULL,
    deployment VARCHAR(50) NOT NULL UNIQUE,
    start_at TIMESTAMPTZ(6) NOT NULL DEFAULT NOW(),

    PRIMARY KEY (id)
);

