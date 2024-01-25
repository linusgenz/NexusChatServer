const sqlite3 = require("sqlite3")
const db = new sqlite3.Database("./data/data.sqlite");

db.run(`
    CREATE TABLE IF NOT EXISTS servers (
        server_id INTEGER PRIMARY KEY,
        server_name TEXT NOT NULL,
        img TEXT,
        created_at DATETIME DEFAULT CURRENT_TIMESTAMP
    )
`);

db.run(`
    CREATE TABLE IF NOT EXISTS users (
        user_id INTEGER PRIMARY KEY,
        username TEXT NOT NULL,
        display_name TEXT
        password TEXT,
        email TEXT,
        language TEXT,
        status INTEGER,
        bio TEXT,
        custom_status TEXT,
        last_seen DATETIME DEFAULT CURRENT_TIMESTAMP,
        joined_at DATETIME DEFAULT CURRENT_TIMESTAMP,
        pronouns TEXT,
        img_url TEXT,
        password TEXT
    )
`);

db.run(`
    CREATE TABLE IF NOT EXISTS server_members (
        membership_id INTEGER PRIMARY KEY,
        server_id INTEGER,
        user_id INTEGER,
        server_owner BOOLEAN,
        joined_at DATETIME DEFAULT CURRENT_TIMESTAMP,
        FOREIGN KEY (server_id) REFERENCES servers(server_id),
        FOREIGN KEY (user_id) REFERENCES users(user_id)
    )
`);

db.run(`
    CREATE TABLE IF NOT EXISTS channels (
        channel_id INTEGER PRIMARY KEY,
        server_id INTEGER,
        type INTEGER,
        channel_name TEXT NOT NULL,
        created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
        FOREIGN KEY (server_id) REFERENCES servers(server_id)
    )
`);

db.run(`
    CREATE TABLE IF NOT EXISTS messages (
        message_id INTEGER PRIMARY KEY,
        channel_id INTEGER,
        user_id INTEGER,
        message_text TEXT NOT NULL,
        sent_at DATETIME DEFAULT CURRENT_TIMESTAMP,
        FOREIGN KEY (channel_id) REFERENCES channels(channel_id),
        FOREIGN KEY (user_id) REFERENCES users(user_id)
    )
`);

db.close((err) => {
	if (err) {
		return console.error(err.message);
	}
	console.log("Database initialization complete. Chat platform database created.");
});
