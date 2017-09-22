CREATE TABLE files (
    url        VARCHAR(255) PRIMARY KEY,
    hash       VARCHAR(32),
    resolution VARCHAR(50),
    bitrate    VARCHAR(50)
);

CREATE TABLE log (
    url     VARCHAR(255) NOT NULL,
    status  INTEGER NOT NULL DEFAULT -1,
    message VARCHAR(300) DEFAULT ''
);
