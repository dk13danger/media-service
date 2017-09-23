CREATE TABLE files (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    url        VARCHAR(255) NOT NULL,
    hash       VARCHAR(32)  NOT NULL,
    resolution VARCHAR(20) DEFAULT '',
    bitrate    VARCHAR(20) DEFAULT ''
);

CREATE TABLE log (
    id      INTEGER PRIMARY KEY AUTOINCREMENT,
    file_id INTEGER NOT NULL,
    status  INTEGER NOT NULL,
    message VARCHAR(300) NOT NULL
);

CREATE UNIQUE INDEX idx_files_url_hash ON files (url, hash);
