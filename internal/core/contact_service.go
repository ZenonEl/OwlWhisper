package core

import (
	"context"
	"fmt"
	"log"
	"time"

	"OwlWhisper/pkg/interfaces"

	"github.com/libp2p/go-libp2p/core/peer"
)

// ContactService реализует IContactService интерфейс
type ContactService struct {
	contactRepo interfaces.IContactRepository
	transport   interfaces.ITransport
}

// NewContactService создает новый экземпляр ContactService
func NewContactService(contactRepo interfaces.IContactRepository, transport interfaces.ITransport) *ContactService {
	return &ContactService{
		contactRepo: contactRepo,
		transport:   transport,
	}
}

// AddContact добавляет новый контакт
func (s *ContactService) AddContact(ctx context.Context, peerID peer.ID, nickname string) error {
	// Проверяем, не пустое ли имя
	if nickname == "" {
		nickname = peerID.ShortString() // Используем короткий PeerID как имя по умолчанию
	}

	// Проверяем, не существует ли уже контакт
	existingContact, err := s.contactRepo.GetContact(ctx, peerID.String())
	if err != nil {
		return fmt.Errorf("failed to check existing contact: %w", err)
	}

	if existingContact != nil {
		return fmt.Errorf("contact with peer ID %s already exists", peerID.ShortString())
	}

	// Создаем новый контакт
	contact := &interfaces.Contact{
		PeerID:   peerID.String(),
		Nickname: nickname,
		AddedAt:  time.Now(),
		LastSeen: time.Now(),
		IsOnline: false,
	}

	// Сохраняем контакт
	if err := s.contactRepo.SaveContact(ctx, contact); err != nil {
		return fmt.Errorf("failed to save contact: %w", err)
	}

	log.Printf("✅ Контакт добавлен: %s (%s)", nickname, peerID.ShortString())
	return nil
}

// RemoveContact удаляет контакт
func (s *ContactService) RemoveContact(ctx context.Context, peerID peer.ID) error {
	// Проверяем, существует ли контакт
	existingContact, err := s.contactRepo.GetContact(ctx, peerID.String())
	if err != nil {
		return fmt.Errorf("failed to check existing contact: %w", err)
	}

	if existingContact == nil {
		return fmt.Errorf("contact with peer ID %s not found", peerID.ShortString())
	}

	// Удаляем контакт
	if err := s.contactRepo.DeleteContact(ctx, peerID.String()); err != nil {
		return fmt.Errorf("failed to delete contact: %w", err)
	}

	log.Printf("🗑️ Контакт удален: %s", peerID.ShortString())
	return nil
}

// GetContacts возвращает все контакты
func (s *ContactService) GetContacts(ctx context.Context) ([]*interfaces.Contact, error) {
	contacts, err := s.contactRepo.GetAllContacts(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get contacts: %w", err)
	}

	return contacts, nil
}

// UpdateContact обновляет информацию о контакте
func (s *ContactService) UpdateContact(ctx context.Context, contact *interfaces.Contact) error {
	// Проверяем, существует ли контакт
	existingContact, err := s.contactRepo.GetContact(ctx, contact.PeerID)
	if err != nil {
		return fmt.Errorf("failed to check existing contact: %w", err)
	}

	if existingContact == nil {
		return fmt.Errorf("contact with peer ID %s not found", contact.PeerID)
	}

	// Обновляем контакт
	if err := s.contactRepo.UpdateContact(ctx, contact); err != nil {
		return fmt.Errorf("failed to update contact: %w", err)
	}

	log.Printf("✏️ Контакт обновлен: %s", contact.PeerID)
	return nil
}

// GetContact возвращает контакт по PeerID
func (s *ContactService) GetContact(ctx context.Context, peerID peer.ID) (*interfaces.Contact, error) {
	contact, err := s.contactRepo.GetContact(ctx, peerID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to get contact: %w", err)
	}

	return contact, nil
}

// UpdateLastSeen обновляет время последнего появления пира
func (s *ContactService) UpdateLastSeen(ctx context.Context, peerID peer.ID) error {
	// Получаем текущий контакт
	contact, err := s.contactRepo.GetContact(ctx, peerID.String())
	if err != nil {
		return fmt.Errorf("failed to get contact: %w", err)
	}

	if contact == nil {
		// Если контакт не существует, создаем его с именем по умолчанию
		contact = &interfaces.Contact{
			PeerID:   peerID.String(),
			Nickname: peerID.ShortString(),
			AddedAt:  time.Now(),
			LastSeen: time.Now(),
			IsOnline: true,
		}

		if err := s.contactRepo.SaveContact(ctx, contact); err != nil {
			return fmt.Errorf("failed to create contact: %w", err)
		}
	} else {
		// Обновляем время последнего появления
		contact.LastSeen = time.Now()
		contact.IsOnline = true

		if err := s.contactRepo.UpdateContact(ctx, contact); err != nil {
			return fmt.Errorf("failed to update contact: %w", err)
		}
	}

	return nil
}

// SetOffline отмечает контакт как оффлайн
func (s *ContactService) SetOffline(ctx context.Context, peerID peer.ID) error {
	contact, err := s.contactRepo.GetContact(ctx, peerID.String())
	if err != nil {
		return fmt.Errorf("failed to get contact: %w", err)
	}

	if contact != nil {
		contact.IsOnline = false
		if err := s.contactRepo.UpdateContact(ctx, contact); err != nil {
			return fmt.Errorf("failed to update contact: %w", err)
		}
	}

	return nil
}
