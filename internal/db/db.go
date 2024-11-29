package db

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"time"
	"xyz-task-5/internal/models"

	_ "github.com/lib/pq"
)

var DB *sql.DB

func InitDB() {
	dbURI := os.Getenv("DB_URI")
	var err error

	DB, err = sql.Open("postgres", dbURI)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	err = DB.Ping()
	if err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}
	SetupSchema()
	fmt.Println("Database connected successfully!")
}

func SetupSchema() {
	query := `
		-- Users Table
		CREATE TABLE IF NOT EXISTS users (
			id VARCHAR(50) PRIMARY KEY,
			username VARCHAR(100) NOT NULL,
			real_name VARCHAR(200),
			email VARCHAR(200) UNIQUE,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		);

		-- Channels Table
		CREATE TABLE IF NOT EXISTS channels (
			id VARCHAR(50) PRIMARY KEY,
			name VARCHAR(200) NOT NULL,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		);

		-- Main Messages Table
		CREATE TABLE IF NOT EXISTS main_messages (
			id SERIAL PRIMARY KEY,
			slack_message_id VARCHAR(100) UNIQUE,
			channel_id VARCHAR(50) REFERENCES channels(id),
			user_id VARCHAR(50) REFERENCES users(id),
			text TEXT,
			timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
			reply_count INTEGER DEFAULT 0,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		);

		-- Reply Messages Table
		CREATE TABLE IF NOT EXISTS reply_messages (
			id SERIAL PRIMARY KEY,
			slack_message_id VARCHAR(100) UNIQUE,
			main_message_id INTEGER REFERENCES main_messages(id),
			channel_id VARCHAR(50) REFERENCES channels(id),
			user_id VARCHAR(50) REFERENCES users(id),
			text TEXT,
			timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		);

		-- Indexes for performance
		CREATE INDEX IF NOT EXISTS idx_main_messages_channel ON main_messages(channel_id);
		CREATE INDEX IF NOT EXISTS idx_main_messages_user ON main_messages(user_id);
		CREATE INDEX IF NOT EXISTS idx_reply_messages_main ON reply_messages(main_message_id);
		CREATE INDEX IF NOT EXISTS idx_reply_messages_channel ON reply_messages(channel_id);
		CREATE INDEX IF NOT EXISTS idx_reply_messages_user ON reply_messages(user_id);
	`

	_, err := DB.Exec(query)
	if err != nil {
		log.Fatalf("Failed to set up schema: %v", err)
	}
	fmt.Println("Database schema setup complete!")
}

func InsertUser(user models.UserInfo) error {
	query := `
		INSERT INTO users (id, username, real_name, email) 
		VALUES ($1, $2, $3, $4) 
		ON CONFLICT (id) DO UPDATE 
		SET username = EXCLUDED.username, 
			real_name = EXCLUDED.real_name, 
			email = EXCLUDED.email
	`
	_, err := DB.Exec(query, user.ID, user.Name, user.RealName, user.Email)
	return err
}

func InsertChannel(channel struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}) error {
	query := `
		INSERT INTO channels (id, name) 
		VALUES ($1, $2) 
		ON CONFLICT (id) DO NOTHING
	`
	_, err := DB.Exec(query, channel.ID, channel.Name)
	return err
}

func InsertMainMessage(msg models.LogMessage, channelID string, userID string) (int, error) {
	query := `
		INSERT INTO main_messages (slack_message_id, channel_id, user_id, text, timestamp) 
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (slack_message_id) DO NOTHING
		RETURNING id
	`
	timestamp, _ := time.Parse(time.RFC3339, msg.Timestamp)
	var messageID int
	err := DB.QueryRow(query, msg.Timestamp, channelID, userID, msg.Text, timestamp).Scan(&messageID)

	if err == sql.ErrNoRows {
		return 0, nil
	}
	return messageID, err
}

func InsertReplyMessage(reply models.LogMessage, mainMessageID int, channelID string, userID string) error {
	query := `
		INSERT INTO reply_messages (slack_message_id, main_message_id, channel_id, user_id, text, timestamp) 
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (slack_message_id) DO NOTHING
	`
	timestamp, _ := time.Parse(time.RFC3339, reply.Timestamp)
	_, err := DB.Exec(query, reply.Timestamp, mainMessageID, channelID, userID, reply.Text, timestamp)
	return err
}
