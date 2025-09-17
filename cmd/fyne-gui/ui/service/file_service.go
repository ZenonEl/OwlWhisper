// Путь: cmd/fyne-gui/services/file_service.go

package services

import (
	"crypto/sha256"
	"fmt"
	"io"
	"log"
	"os"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"

	newcore "OwlWhisper/cmd/fyne-gui/new-core"
	protocol "OwlWhisper/cmd/fyne-gui/new-core/protocol"
)

type FileCardGenerator interface {
	NewFileCard(metadata *protocol.FileMetadata, onDownload func(*protocol.FileMetadata)) fyne.CanvasObject
}

// FileCard - это кастомный виджет для отображения анонса файла.
type FileCard struct {
	widget.BaseWidget // Встраиваем базовый виджет Fyne
	metadata          *protocol.FileMetadata
	onDownload        func(metadata *protocol.FileMetadata)
}

// NewFileCard создает новый экземпляр FileCard.
func NewFileCard(metadata *protocol.FileMetadata, onDownload func(*protocol.FileMetadata)) *FileCard {
	card := &FileCard{
		metadata:   metadata,
		onDownload: onDownload,
	}
	// ExtendBaseWidget сообщает Fyne, что эта структура является виджетом.
	card.ExtendBaseWidget(card)
	return card
}

// CreateRenderer - ЭТОТ МЕТОД МЫ ЗАБЫЛИ.
// Он вызывается Fyne один раз, чтобы создать "рендерер" для нашего виджета.
// Рендерер - это то, что на самом деле рисует и управляет дочерними элементами.
func (c *FileCard) CreateRenderer() fyne.WidgetRenderer {
	// Создаем все элементы, из которых состоит наша карточка
	filename := widget.NewLabel(c.metadata.Filename)
	filename.TextStyle.Bold = true

	sizeMB := float64(c.metadata.SizeBytes) / 1024.0 / 1024.0
	sizeLabel := widget.NewLabel(fmt.Sprintf("%.2f MB", sizeMB))

	downloadButton := widget.NewButton("Скачать", func() {
		// При нажатии на кнопку вызываем наш callback
		if c.onDownload != nil {
			c.onDownload(c.metadata)
		}
	})

	// Собираем все в контейнер
	content := container.NewVBox(filename, sizeLabel, downloadButton)

	// Возвращаем простой рендерер, который будет управлять этим контейнером
	return widget.NewSimpleRenderer(content)
}

// TransferState описывает состояние текущей передачи файла.
type TransferState struct {
	TransferID string
	IsIncoming bool   // true, если мы скачиваем; false, если мы отдаем
	FilePath   string // Путь к файлу на диске
	Metadata   *protocol.FileMetadata
	StreamID   uint64
	Progress   float64 // от 0.0 до 1.0
	Status     string  // "pending", "transferring", "completed", "failed"
}

// FileService управляет всей логикой передачи файлов.
type FileService struct {
	core           newcore.ICoreController
	contactService *ContactService

	// Хранилище активных и завершенных передач
	transfers map[string]*TransferState
	mu        sync.RWMutex

	onUpdate              func(transferID string)        // Callback для обновления UI
	onNewFileAnnouncement func(widget fyne.CanvasObject) // Принимает готовый виджет
}

func NewFileService(core newcore.ICoreController, cs *ContactService, onNewFile func(fyne.CanvasObject)) *FileService {
	return &FileService{
		core:                  core,
		contactService:        cs,
		transfers:             make(map[string]*TransferState),
		onNewFileAnnouncement: onNewFile,
	}
}

// AnnounceFile (Фаза 1: Анонс) - вызывается из UI, когда пользователь выбирает файл.
func (fs *FileService) AnnounceFile(recipientID, filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("не удалось открыть файл: %w", err)
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return fmt.Errorf("не удалось получить информацию о файле: %w", err)
	}

	// Вычисляем хеш
	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return fmt.Errorf("не удалось вычислить хеш: %w", err)
	}
	fileHash := fmt.Sprintf("%x", hasher.Sum(nil))

	// Создаем метаданные
	metadata := &protocol.FileMetadata{
		TransferId: uuid.New().String(),
		Filename:   stat.Name(),
		SizeBytes:  stat.Size(),
		HashSha256: fileHash,
		// MimeType можно определять с помощью доп. библиотек, пока оставим пустым
	}

	// Сохраняем состояние передачи у себя
	state := &TransferState{
		TransferID: metadata.TransferId,
		IsIncoming: false, // Мы - отправитель
		FilePath:   filePath,
		Metadata:   metadata,
		Status:     "announced",
	}
	fs.mu.Lock()
	fs.transfers[metadata.TransferId] = state
	fs.mu.Unlock()

	// Упаковываем в Envelope
	chatMsg := &protocol.ChatMessage{
		ChatType: protocol.ChatMessage_PRIVATE,
		ChatId:   recipientID,
		Content:  &protocol.ChatMessage_FileAnnouncement{FileAnnouncement: metadata},
	}
	envelope := &protocol.Envelope{
		MessageId:     uuid.New().String(),
		SenderId:      fs.core.GetMyPeerID(),
		TimestampUnix: time.Now().Unix(),
		Payload:       &protocol.Envelope_ChatMessage{ChatMessage: chatMsg},
	}

	data, err := proto.Marshal(envelope)
	if err != nil {
		return err
	}

	log.Printf("INFO: [FileService] Анонсируем файл %s пиру %s", metadata.Filename, recipientID[:8])
	return fs.core.SendDataToPeer(recipientID, data)
}

// HandleDownloadRequest (Фаза 4: Раздача) - вызывается из Dispatcher'а.
func (fs *FileService) HandleDownloadRequest(req *protocol.FileDownloadRequest, senderID string) {
	fs.mu.Lock()
	state, ok := fs.transfers[req.TransferId]
	if !ok {
		log.Printf("WARN: [FileService] Получен запрос на скачивание неизвестного transferID: %s", req.TransferId)
		// TODO: Отправить FileUnavailable
		fs.mu.Unlock()
		return
	}
	fs.mu.Unlock()

	log.Printf("INFO: [FileService] Получен запрос на скачивание файла %s от %s", state.Metadata.Filename, senderID[:8])

	// Запускаем процесс передачи в новой горутине
	go func() {
		// 1. Открываем новый стрим для файла
		streamID, err := fs.core.OpenStream(senderID, newcore.FILE_PROTOCOL_ID)
		if err != nil {
			log.Printf("ERROR: [FileService] Не удалось открыть файловый стрим: %v", err)
			return
		}

		state.StreamID = streamID
		state.Status = "transferring"
		fs.onUpdate(state.TransferID)

		// 2. Открываем файл для чтения
		file, err := os.Open(state.FilePath)
		if err != nil {
			log.Printf("ERROR: [FileService] Не удалось открыть файл для отправки: %v", err)
			return
		}
		defer file.Close()

		// 3. Читаем и отправляем "кусками"
		buffer := make([]byte, 65536)
		for {
			n, err := file.Read(buffer)
			if err == io.EOF {
				break
			}
			if err != nil { /* обработка ошибки */
				return
			}

			if err := fs.core.WriteToStream(streamID, buffer[:n]); err != nil {
				log.Printf("ERROR: [FileService] Ошибка записи в файловый стрим: %v", err)
				return
			}
		}

		// 4. Закрываем стрим, когда все отправлено
		fs.core.CloseStream(streamID)
		state.Status = "completed"
		fs.onUpdate(state.TransferID)
		log.Printf("INFO: [FileService] Передача файла %s завершена.", state.Metadata.Filename)
	}()
}

// HandleFileAnnouncement (Фаза 2: Получение анонса) - вызывается из Dispatcher'а.
func (fs *FileService) HandleFileAnnouncement(senderID string, metadata *protocol.FileMetadata) {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	// Проверяем, не получали ли мы уже анонс этого файла
	if _, ok := fs.transfers[metadata.TransferId]; ok {
		return // Уже знаем об этой передаче
	}

	// Сохраняем состояние передачи. Мы - получатель.
	state := &TransferState{
		TransferID: metadata.TransferId,
		IsIncoming: true,
		FilePath:   "", // Мы еще не знаем, куда сохранять файл
		Metadata:   metadata,
		StreamID:   0,
		Progress:   0,
		Status:     "announced", // Новый статус "анонсировано"
	}

	fs.transfers[metadata.TransferId] = state
	log.Printf("INFO: [FileService] Получен анонс файла '%s' от %s. Готов к скачиванию.", metadata.Filename, senderID[:8])

	// Уведомляем UI, что нужно нарисовать плашку с файлом
	// 1. Создаем виджет FileCard
	fileCard := NewFileCard(metadata, func(m *protocol.FileMetadata) {
		// Это обработчик нажатия кнопки "Скачать"
		go fs.RequestFileDownload(m, senderID)
	})
	// 2. Вызываем callback, чтобы передать ГОТОВЫЙ ВИДЖЕТ в UI
	fs.onNewFileAnnouncement(fileCard)
}

// RequestFileDownload (Фаза 3: Инициация скачивания) - вызывается из UI.

func (fs *FileService) RequestFileDownload(metadata *protocol.FileMetadata, senderID string) {
	log.Printf("INFO: [FileService] Запрашиваем скачивание файла %s от %s", metadata.Filename, senderID[:8])

	// 1. Создаем сам запрос
	req := &protocol.FileDownloadRequest{
		TransferId: metadata.TransferId,
	}

	// 2. Оборачиваем его в ChatMessage, так как это событие чата
	chatMsg := &protocol.ChatMessage{
		ChatType: protocol.ChatMessage_PRIVATE,
		ChatId:   senderID,
		Content:  &protocol.ChatMessage_FileRequest{FileRequest: req}, // <-- ИСПОЛЬЗУЕМ ПРАВИЛЬНЫЙ ТИП
	}

	// 3. Оборачиваем ChatMessage в Envelope
	envelope := &protocol.Envelope{
		MessageId:     uuid.New().String(),
		SenderId:      fs.core.GetMyPeerID(),
		TimestampUnix: time.Now().Unix(),
		Payload:       &protocol.Envelope_ChatMessage{ChatMessage: chatMsg},
	}

	data, err := proto.Marshal(envelope)
	if err != nil {
		log.Printf("ERROR: [FileService] Ошибка Marshal при создании DownloadRequest: %v", err)
		return
	}

	if err := fs.core.SendDataToPeer(senderID, data); err != nil {
		log.Printf("ERROR: [FileService] Не удалось отправить DownloadRequest: %v", err)
	}
}

// HandleIncomingStream - вызывается из Core, когда к нам открыли файловый стрим.
func (fs *FileService) HandleIncomingStream(payload newcore.NewIncomingStreamPayload) {
	// ... (здесь будет логика связывания streamID с transferID) ...
}

// HandleStreamData - вызывается из Core, когда пришли "куски" файла.
func (fs *FileService) HandleStreamData(payload newcore.StreamDataReceivedPayload) {
	// ... (здесь будет логика записи байтов на диск и обновления прогресс-бара) ...
}
