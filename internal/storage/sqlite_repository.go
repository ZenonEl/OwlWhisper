package storage

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"OwlWhisper/pkg/interfaces"

	_ "github.com/mattn/go-sqlite3"
)

// SQLiteRepository реализует интерфейсы IMessageRepository и IContactRepository
type SQLiteRepository struct {
	db *sql.DB
}

// NewSQLiteRepository создает новый экземпляр SQLiteRepository
func NewSQLiteRepository(dbPath string) (*SQLiteRepository, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	repo := &SQLiteRepository{db: db}

	// Инициализируем таблицы
	if err := repo.initTables(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize tables: %w", err)
	}

	return repo, nil
}

// Close закрывает соединение с базой данных
func (r *SQLiteRepository) Close() error {
	return r.db.Close()
}

// initTables создает необходимые таблицы
func (r *SQLiteRepository) initTables() error {
	// Таблица сообщений
	createMessagesTable := `
	CREATE TABLE IF NOT EXISTS messages (
		id TEXT PRIMARY KEY,
		from_peer TEXT NOT NULL,
		to_peer TEXT NOT NULL,
		content TEXT NOT NULL,
		timestamp DATETIME NOT NULL,
		type TEXT NOT NULL DEFAULT 'text',
		is_read BOOLEAN NOT NULL DEFAULT FALSE,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	`

	// Таблица контактов
	createContactsTable := `
	CREATE TABLE IF NOT EXISTS contacts (
		peer_id TEXT PRIMARY KEY,
		nickname TEXT NOT NULL,
		added_at DATETIME NOT NULL,
		last_seen DATETIME,
		is_online BOOLEAN NOT NULL DEFAULT FALSE,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	`

	// Создаем таблицы
	if _, err := r.db.Exec(createMessagesTable); err != nil {
		return fmt.Errorf("failed to create messages table: %w", err)
	}

	if _, err := r.db.Exec(createContactsTable); err != nil {
		return fmt.Errorf("failed to create contacts table: %w", err)
	}

	// Создаем индексы для оптимизации
	createIndexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_messages_from_peer ON messages(from_peer);",
		"CREATE INDEX IF NOT EXISTS idx_messages_to_peer ON messages(to_peer);",
		"CREATE INDEX IF NOT EXISTS idx_messages_timestamp ON messages(timestamp);",
		"CREATE INDEX IF NOT EXISTS idx_messages_conversation ON messages(from_peer, to_peer, timestamp);",
		"CREATE INDEX IF NOT EXISTS idx_contacts_peer_id ON contacts(peer_id);",
		"CREATE INDEX IF NOT EXISTS idx_contacts_last_seen ON contacts(last_seen);",
	}

	for _, index := range createIndexes {
		if _, err := r.db.Exec(index); err != nil {
			log.Printf("Warning: failed to create index: %v", err)
		}
	}

	return nil
}

// SaveMessage сохраняет сообщение
func (r *SQLiteRepository) SaveMessage(ctx context.Context, message *interfaces.Message) error {
	query := `
	INSERT OR REPLACE INTO messages (id, from_peer, to_peer, content, timestamp, type)
	VALUES (?, ?, ?, ?, ?, ?)
	`

	_, err := r.db.ExecContext(ctx, query,
		message.ID,
		message.FromPeer,
		message.ToPeer,
		message.Content,
		message.Timestamp,
		message.Type,
	)

	if err != nil {
		return fmt.Errorf("failed to save message: %w", err)
	}

	return nil
}

// GetMessages возвращает сообщения между двумя пирами
func (r *SQLiteRepository) GetMessages(ctx context.Context, peer1, peer2 string, limit, offset int) ([]*interfaces.Message, error) {
	query := `
	SELECT id, from_peer, to_peer, content, timestamp, type
	FROM messages
	WHERE (from_peer = ? AND to_peer = ?) OR (from_peer = ? AND to_peer = ?)
	ORDER BY timestamp DESC
	LIMIT ? OFFSET ?
	`

	rows, err := r.db.QueryContext(ctx, query, peer1, peer2, peer2, peer1, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query messages: %w", err)
	}
	defer rows.Close()

	var messages []*interfaces.Message
	for rows.Next() {
		msg := &interfaces.Message{}
		err := rows.Scan(&msg.ID, &msg.FromPeer, &msg.ToPeer, &msg.Content, &msg.Timestamp, &msg.Type)
		if err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}
		messages = append(messages, msg)
	}

	// Разворачиваем порядок, чтобы старые сообщения были первыми
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	return messages, nil
}

// GetLastMessage возвращает последнее сообщение между двумя пирами
func (r *SQLiteRepository) GetLastMessage(ctx context.Context, peer1, peer2 string) (*interfaces.Message, error) {
	query := `
	SELECT id, from_peer, to_peer, content, timestamp, type
	FROM messages
	WHERE (from_peer = ? AND to_peer = ?) OR (from_peer = ? AND to_peer = ?)
	ORDER BY timestamp DESC
	LIMIT 1
	`

	msg := &interfaces.Message{}
	err := r.db.QueryRowContext(ctx, query, peer1, peer2, peer2, peer1).Scan(
		&msg.ID, &msg.FromPeer, &msg.ToPeer, &msg.Content, &msg.Timestamp, &msg.Type,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get last message: %w", err)
	}

	return msg, nil
}

// DeleteMessage удаляет сообщение по ID
func (r *SQLiteRepository) DeleteMessage(ctx context.Context, messageID string) error {
	query := "DELETE FROM messages WHERE id = ?"

	result, err := r.db.ExecContext(ctx, query, messageID)
	if err != nil {
		return fmt.Errorf("failed to delete message: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("message with id %s not found", messageID)
	}

	return nil
}

// GetUnreadCount возвращает количество непрочитанных сообщений
func (r *SQLiteRepository) GetUnreadCount(ctx context.Context, peerID string) (int, error) {
	query := `
	SELECT COUNT(*)
	FROM messages
	WHERE to_peer = ? AND is_read = FALSE
	`

	var count int
	err := r.db.QueryRowContext(ctx, query, peerID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get unread count: %w", err)
	}

	return count, nil
}

// MarkAsRead отмечает сообщения как прочитанные
func (r *SQLiteRepository) MarkAsRead(ctx context.Context, peerID string) error {
	query := `
	UPDATE messages
	SET is_read = TRUE
	WHERE to_peer = ? AND is_read = FALSE
	`

	_, err := r.db.ExecContext(ctx, query, peerID)
	if err != nil {
		return fmt.Errorf("failed to mark messages as read: %w", err)
	}

	return nil
}

// SaveContact сохраняет контакт
func (r *SQLiteRepository) SaveContact(ctx context.Context, contact *interfaces.Contact) error {
	query := `
	INSERT OR REPLACE INTO contacts (peer_id, nickname, added_at, last_seen, is_online)
	VALUES (?, ?, ?, ?, ?)
	`

	_, err := r.db.ExecContext(ctx, query,
		contact.PeerID,
		contact.Nickname,
		contact.AddedAt,
		contact.LastSeen,
		contact.IsOnline,
	)

	if err != nil {
		return fmt.Errorf("failed to save contact: %w", err)
	}

	return nil
}

// GetContact возвращает контакт по PeerID
func (r *SQLiteRepository) GetContact(ctx context.Context, peerID string) (*interfaces.Contact, error) {
	query := `
	SELECT peer_id, nickname, added_at, last_seen, is_online
	FROM contacts
	WHERE peer_id = ?
	`

	contact := &interfaces.Contact{}
	err := r.db.QueryRowContext(ctx, query, peerID).Scan(
		&contact.PeerID, &contact.Nickname, &contact.AddedAt, &contact.LastSeen, &contact.IsOnline,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get contact: %w", err)
	}

	return contact, nil
}

// GetAllContacts возвращает все контакты
func (r *SQLiteRepository) GetAllContacts(ctx context.Context) ([]*interfaces.Contact, error) {
	query := `
	SELECT peer_id, nickname, added_at, last_seen, is_online
	FROM contacts
	ORDER BY nickname ASC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query contacts: %w", err)
	}
	defer rows.Close()

	var contacts []*interfaces.Contact
	for rows.Next() {
		contact := &interfaces.Contact{}
		err := rows.Scan(&contact.PeerID, &contact.Nickname, &contact.AddedAt, &contact.LastSeen, &contact.IsOnline)
		if err != nil {
			return nil, fmt.Errorf("failed to scan contact: %w", err)
		}
		contacts = append(contacts, contact)
	}

	return contacts, nil
}

// UpdateContact обновляет информацию о контакте
func (r *SQLiteRepository) UpdateContact(ctx context.Context, contact *interfaces.Contact) error {
	query := `
	UPDATE contacts
	SET nickname = ?, last_seen = ?, is_online = ?, updated_at = CURRENT_TIMESTAMP
	WHERE peer_id = ?
	`

	result, err := r.db.ExecContext(ctx, query,
		contact.Nickname,
		contact.LastSeen,
		contact.IsOnline,
		contact.PeerID,
	)

	if err != nil {
		return fmt.Errorf("failed to update contact: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("contact with peer_id %s not found", contact.PeerID)
	}

	return nil
}

// DeleteContact удаляет контакт
func (r *SQLiteRepository) DeleteContact(ctx context.Context, peerID string) error {
	query := "DELETE FROM contacts WHERE peer_id = ?"

	result, err := r.db.ExecContext(ctx, query, peerID)
	if err != nil {
		return fmt.Errorf("failed to delete contact: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("contact with peer_id %s not found", peerID)
	}

	return nil
}

// UpdateLastSeen обновляет время последнего появления пира
func (r *SQLiteRepository) UpdateLastSeen(ctx context.Context, peerID string, lastSeen time.Time) error {
	query := `
	UPDATE contacts
	SET last_seen = ?, updated_at = CURRENT_TIMESTAMP
	WHERE peer_id = ?
	`

	_, err := r.db.ExecContext(ctx, query, lastSeen, peerID)
	if err != nil {
		return fmt.Errorf("failed to update last seen: %w", err)
	}

	return nil
}
