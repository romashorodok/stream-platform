
CREATE TABLE users (
    id UUID NOT NULL DEFAULT uuid_generate_v4(),
    username varchar(30) NOT NULL UNIQUE,
    password varchar(200) NOT NULL,

    PRIMARY KEY (id)
);

