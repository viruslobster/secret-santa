CREATE TABLE chats (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    thread TEXT NOT NULL,
    message TEXT NOT NULL,
    user_id INTEGER NOT NULL,
    timestamp INTEGER NOT NULL
);
CREATE INDEX chats_thread ON chats (thread);

CREATE TABLE users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    credentials TEXT NOT NULL
);

CREATE TABLE servers (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    names TEXT NOT NULL,
    pairings TEXT NOT NULL
);

CREATE TABLE server_members (
    server_id INTEGER NOT NULL REFERENCES servers(id),
    user_id INTEGER NOT NULL REFERENCES users(id),

    -- Index into servers.names
    name_index INTEGER NOT NULL,
    PRIMARY KEY (server_id, name_index)
);

CREATE TABLE threads (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    server_id INTEGER REFERENCES servers(id),
    -- Index into servers.names, used to link this thread with a pairing
    name_index INTEGER NOT NULL,
    santa INTEGER REFERENCES users(id),
    giftee INTEGER REFERENCES users(id)
);
