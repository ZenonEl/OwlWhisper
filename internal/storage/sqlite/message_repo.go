package sqlite

import (
	"database/sql"
	"fmt"
	"time"

	"OwlWhisper/internal/protocol/protocol"
	"OwlWhisper/internal/storage"

	_ "github.com/mattn/go-sqlite3"
)

// MessageRepository реализует IMessageRepository для SQLite
type MessageRepository struct {
	db *sql.DB
}

// NewMessageRepository создает новый репозиторий сообщений
func NewMessageRepository(dbPath string) (*MessageRepository, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("не удалось открыть базу данных: %w", err)
	}

	repo := &MessageRepository{db: db}

	// Создаем таблицу если не существует
	if err := repo.createTable(); err != nil {
		db.Close()
		return nil, fmt.Errorf("не удалось создать таблицу: %w", err)
	}

	return repo, nil
}

// createTable создает таблицу сообщений
func (r *MessageRepository) createTable() error {
	query := `
	CREATE TABLE IF NOT EXISTS messages (
		id TEXT PRIMARY KEY,
		text TEXT NOT NULL,
		timestamp_unix INTEGER NOT NULL,
		sender_id TEXT NOT NULL,
		chat_type TEXT NOT NULL,
		recipient_id TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	
	CREATE INDEX IF NOT EXISTS idx_messages_sender_id ON messages(sender_id);
	CREATE INDEX IF NOT EXISTS idx_messages_timestamp ON messages(timestamp_unix);
	CREATE INDEX IF NOT EXISTS idx_messages_chat_type ON messages(chat_type);
	`

	_, err := r.db.Exec(query)
	return err
}

// Save сохраняет сообщение в базу данных
func (r *MessageRepository) Save(message *protocol.ChatMessage, senderID string) error {
	query := `
	INSERT INTO messages (id, text, timestamp_unix, sender_id, chat_type, recipient_id, created_at, updated_at)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	now := time.Now()
	_, err := r.db.Exec(query,
		message.MessageId,
		message.Text,
		message.TimestampUnix,
		senderID,
		message.ChatType,
		message.RecipientId,
		now,
		now,
	)

	if err != nil {
		return fmt.Errorf("не удалось сохранить сообщение: %w", err)
	}

	return nil
}

// GetHistory возвращает историю сообщений
func (r *MessageRepository) GetHistory(limit int) ([]storage.StoredMessage, error) {
	query := `
	SELECT id, text, timestamp_unix, sender_id, chat_type, recipient_id, created_at, updated_at
	FROM messages
	ORDER BY timestamp_unix DESC
	LIMIT ?
	`

	rows, err := r.db.Query(query, limit)
	if err != nil {
		return nil, fmt.Errorf("не удалось получить историю: %w", err)
	}
	defer rows.Close()

	var messages []storage.StoredMessage
	for rows.Next() {
		var msg storage.StoredMessage
		var timestampUnix int64

		err := rows.Scan(
			&msg.ID,
			&msg.Text,
			&timestampUnix,
			&msg.SenderID,
			&msg.ChatType,
			&msg.RecipientID,
			&msg.CreatedAt,
			&msg.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("ошибка сканирования строки: %w", err)
		}

		msg.Timestamp = time.Unix(timestampUnix, 0)
		messages = append(messages, msg)
	}

	return messages, nil
}

// GetMessagesByPeer возвращает сообщения от конкретного пира
func (r *MessageRepository) GetMessagesByPeer(peerID string, limit int) ([]storage.StoredMessage, error) {
	query := `
	SELECT id, text, timestamp_unix, sender_id, chat_type, recipient_id, created_at, updated_at
	FROM messages
	WHERE sender_id = ?
	ORDER BY timestamp_unix DESC
	LIMIT ?
	`

	rows, err := r.db.Query(query, peerID, limit)
	if err != nil {
		return nil, fmt.Errorf("не удалось получить сообщения от пира: %w", err)
	}
	defer rows.Close()

	var messages []storage.StoredMessage
	for rows.Next() {
		var msg storage.StoredMessage
		var timestampUnix int64

		err := rows.Scan(
			&msg.ID,
			&msg.Text,
			&timestampUnix,
			&msg.SenderID,
			&msg.ChatType,
			&msg.RecipientID,
			&msg.CreatedAt,
			&msg.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("ошибка сканирования строки: %w", err)
		}

		msg.Timestamp = time.Unix(timestampUnix, 0)
		messages = append(messages, msg)
	}

	return messages, nil
}

// GetMessageByID возвращает сообщение по ID
func (r *MessageRepository) GetMessageByID(messageID string) (*storage.StoredMessage, error) {
	query := `
	SELECT id, text, timestamp_unix, sender_id, chat_type, recipient_id, created_at, updated_at
	FROM messages
	WHERE id = ?
	`

	var msg storage.StoredMessage
	var timestampUnix int64

	err := r.db.QueryRow(query, messageID).Scan(
		&msg.ID,
		&msg.Text,
		&timestampUnix,
		&msg.SenderID,
		&msg.ChatType,
		&msg.RecipientID,
		&msg.CreatedAt,
		&msg.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("сообщение не найдено: %s", messageID)
		}
		return nil, fmt.Errorf("ошибка получения сообщения: %w", err)
	}

	msg.Timestamp = time.Unix(timestampUnix, 0)
	return &msg, nil
}

// DeleteMessage удаляет сообщение по ID
func (r *MessageRepository) DeleteMessage(messageID string) error {
	query := `DELETE FROM messages WHERE id = ?`

	result, err := r.db.Exec(query, messageID)
	if err != nil {
		return fmt.Errorf("не удалось удалить сообщение: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("ошибка получения количества удаленных строк: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("сообщение не найдено: %s", messageID)
	}

	return nil
}

// Close закрывает соединение с базой данных
func (r *MessageRepository) Close() error {
	return r.db.Close()
}
