package storage

import (
	"database/sql"
	"time"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

// Message represents a message from an IRC channel.
type Message struct {
	ID           int       `db:"id"`
	Channel      string    `db:"channel"`
	Timestamp    time.Time `db:"timestamp"`
	Sender       string    `db:"sender"`
	Message      string    `db:"message"`
	Date         string    `db:"date"`
	IRCCloudTime int64     `db:"irccloud_time"`
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
		irccloud_time INTEGER UNIQUE
	);
	`
	_, err := db.Exec(schema)
	return err
}

// InsertMessage inserts a new message into the database.
func (db *DB) InsertMessage(m *Message) error {
	query := `
	INSERT OR IGNORE INTO messages (channel, timestamp, sender, message, date, irccloud_time)
	VALUES (:channel, :timestamp, :sender, :message, :date, :irccloud_time)
	`
	_, err := db.NamedExec(query, m)
	return err
}

// GetMessagesByDate retrieves all messages for a given date.
func (db *DB) GetMessagesByDate(date string) ([]Message, error) {
	var messages []Message
	query := `
	SELECT * FROM messages
	WHERE date = ?
	`
	err := db.Select(&messages, query, date)
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
	_, err := db.Exec(query, date)
	return err
}
