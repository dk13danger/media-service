CREATE TABLE files (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    url        VARCHAR(255) UNIQUE,
    hash       VARCHAR(32),
    resolution VARCHAR(50),
    bitrate    VARCHAR(50)
);

CREATE TABLE log (
    id      INTEGER PRIMARY KEY AUTOINCREMENT,
    file_id INTEGER NOT NULL,
    status  INTEGER NOT NULL DEFAULT -1,
    message VARCHAR(300) DEFAULT ''
);
