
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE active_streams (
    id UUID NOT NULL DEFAULT uuid_generate_v4(),

    broadcaster_id UUID NOT NULL UNIQUE,
    username VARCHAR(30) NOT NULL UNIQUE,

    namespace VARCHAR(50) NOT NULL,
    deployment VARCHAR(50) NOT NULL UNIQUE,
    start_at TIMESTAMPTZ(6) NOT NULL DEFAULT NOW(),

    PRIMARY KEY (id)
);

