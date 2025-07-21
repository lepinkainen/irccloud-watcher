package storage

import (
	"database/sql"
	"time"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

// Message represents a message from an IRC channel.
type Message struct {
	ID        int       `db:"id"`
	Channel   string    `db:"channel"`
	Timestamp time.Time `db:"timestamp"`
	Sender    string    `db:"sender"`
	Message   string    `db:"message"`
	Date      string    `db:"date"`
	EID       int64     `db:"eid"`
}

// DB is a wrapper around sqlx.DB for SQLite operations.
type DB struct {
	*sqlx.DB
}

// NewDB creates a new database connection.
func NewDB(dataSourceName string) (*DB, error) {
	db, err := sqlx.Connect("sqlite", dataSourceName)
	if err != nil {
		return nil, err
	}

	if err := createSchema(db); err != nil {
		return nil, err
	}

	return &DB{db}, nil
}

// createSchema creates the database schema if it doesn't exist.
func createSchema(db *sqlx.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS messages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		channel TEXT NOT NULL,
		timestamp DATETIME NOT NULL,
		sender TEXT,
		message TEXT,
		date DATE NOT NULL,
		eid INTEGER UNIQUE
	);
	
	CREATE INDEX IF NOT EXISTS idx_messages_date ON messages(date);
	CREATE INDEX IF NOT EXISTS idx_messages_channel ON messages(channel);
	CREATE UNIQUE INDEX IF NOT EXISTS idx_messages_eid ON messages(eid);
	`
	_, err := db.Exec(schema)
	if err != nil {
		return err
	}

	// Handle migration for existing databases - add eid column if it doesn't exist
	migrationSchema := `
	ALTER TABLE messages ADD COLUMN eid INTEGER;
	`
	// This will fail silently if the column already exists, which is expected
	_, _ = db.Exec(migrationSchema)

	// Create the unique index if it doesn't exist (will fail silently if exists)
	indexSchema := `
	CREATE UNIQUE INDEX IF NOT EXISTS idx_messages_eid ON messages(eid);
	`
	_, _ = db.Exec(indexSchema)

	return nil
}

// InsertMessage inserts a new message into the database.
func (db *DB) InsertMessage(m *Message) error {
	// Use INSERT OR IGNORE to handle duplicates based on EID uniqueness
	// EID is IRCCloud's unique event identifier, so this is the most reliable deduplication
	query := `
	INSERT OR IGNORE INTO messages (channel, timestamp, sender, message, date, eid)
	VALUES (:channel, :timestamp, :sender, :message, :date, :eid)
	`
	_, err := db.DB.NamedExec(query, m)
	return err
}

// GetMessagesByDate retrieves all messages for a given date.
func (db *DB) GetMessagesByDate(date string) ([]Message, error) {
	var messages []Message
	query := `
	SELECT * FROM messages
	WHERE date = ?
	`
	err := db.DB.Select(&messages, query, date)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return messages, err
}

// DeleteMessagesByDate deletes all messages for a given date.
func (db *DB) DeleteMessagesByDate(date string) error {
	query := `
	DELETE FROM messages
	WHERE date = ?
	`
	_, err := db.DB.Exec(query, date)
	return err
}
