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
    admin_id INTEGER NOT NULL,
    group_thread TEXT NOT NULL,
    names TEXT NOT NULL,
    pairings TEXT NOT NULL,
    user_threads TEXT NOT NULL
);

CREATE TABLE server_members (
    server_id INTEGER NOT NULL REFERENCES servers(id),
    name TEXT NOT NULL,
    user_id INTEGER NOT NULL,
    PRIMARY KEY (server_id, name)
);

CREATE TABLE threads (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    santa INTEGER NOT NULL REFERENCES users(id),
    giftee INTEGER NOT NULL REFERENCES users(id)
);
