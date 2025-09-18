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
	pipeReader *io.PipeReader
	pipeWriter *io.PipeWriter
}

// FileService управляет всей логикой передачи файлов.
type FileService struct {
	core           newcore.ICoreController
	contactService *ContactService
	chatService    *ChatService

	// Хранилище активных и завершенных передач
	transfers map[string]*TransferState
	mu        sync.RWMutex
}

func NewFileService(core newcore.ICoreController, cs *ContactService) *FileService {
func NewFileService(core newcore.ICoreController, cs *ContactService, chs *ChatService) *FileService {
	return &FileService{
		core:           core,
		contactService: cs,
		chatService:    chs,
		transfers:      make(map[string]*TransferState),
	}
}

// AnnounceFile (Фаза 1: Анонс) - вызывается из UI, когда пользователь выбирает файл.
func (fs *FileService) AnnounceFile(recipientID, filePath string) (*FileCard, error) {
	file, err := os.Open(filePath)
	if err != nil {
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
	}

	// Вычисляем хеш
	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
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
		return nil, err
	}

	// ИЗМЕНЕНИЕ: Не вызываем chatService, а создаем и возвращаем виджет
	fileCard := NewFileCard(metadata, func(m *protocol.FileMetadata) {
		// Кнопка "Скачать" для отправленных файлов может ничего не делать
	})

	log.Printf("INFO: [FileService] Анонсируем файл %s пиру %s", metadata.Filename, recipientID[:8])
	err = fs.core.SendDataToPeer(recipientID, data) // Переопределяем err
	if err != nil {
		return nil, err
	}

	return fileCard, nil
}

// HandleDownloadRequest (Фаза 4: Раздача) - вызывается из Dispatcher'а.
func (fs *FileService) HandleDownloadRequest(req *protocol.FileDownloadRequest, senderID string) {
	fs.mu.RLock()
	state, ok := fs.transfers[req.TransferId]
	fs.mu.RUnlock()

	if !ok {
		log.Printf("WARN: [FileService] Получен запрос на скачивание неизвестного transferID: %s", req.TransferId)
		// TODO: Отправить FileTransferStatus{UNAVAILABLE}
		return
	}

	log.Printf("INFO: [FileService] Получен запрос на скачивание файла %s от %s", state.Metadata.Filename, senderID[:8])

	// Запускаем процесс передачи в новой горутине
	go fs.streamFileSender(state, senderID)
}

// streamFileSender - горутина, которая управляет ОТПРАВКОЙ одного файла.
func (fs *FileService) streamFileSender(state *TransferState, recipientID string) {
	// 1. Открываем новый стрим для файла
	streamID, err := fs.core.OpenStream(recipientID, newcore.FILE_PROTOCOL_ID)
	if err != nil {
		log.Printf("ERROR: [FileService] Не удалось открыть файловый стрим: %v", err)
		state.Status = "failed"
		// TODO: Уведомить UI об ошибке
		return
	}
	defer fs.core.CloseStream(streamID)

	state.StreamID = streamID
	state.Status = "transferring"
	// TODO: Уведомить UI, что передача началась

	// 2. Открываем файл для чтения
	file, err := os.Open(state.FilePath)
	if err != nil {
		log.Printf("ERROR: [FileService] Не удалось открыть файл для отправки '%s': %v", state.FilePath, err)
		state.Status = "failed"
		// TODO: Отправить FileTransferStatus с ошибкой
		return
	}
	defer file.Close()

	// 3. Читаем и отправляем "кусками" в виде Protobuf-сообщений
	log.Printf("INFO: [FileService] Начата передача файла %s по стриму %d", state.Metadata.Filename, streamID)
	buffer := make([]byte, 64*1024) // 64KB chunk size
	var offset int64 = 0
	for {
		bytesRead, err := file.Read(buffer)
		if err == io.EOF {
			// Достигли конца файла, выходим из цикла
			break
		}
		if err != nil {
			log.Printf("ERROR: [FileService] Ошибка чтения файла '%s': %v", state.FilePath, err)
			state.Status = "failed"
			// TODO: Отправить FileTransferStatus с ошибкой
			return
		}

		// Создаем Protobuf-сообщение для "куска"
		chunk := &protocol.FileData{
			TransferId: state.TransferID,
			ChunkData:  buffer[:bytesRead],
			Offset:     offset,
			LastChunk:  false,
		}

		// Отправляем "кусок"
		if err := fs.sendChunk(streamID, chunk); err != nil {
			log.Printf("ERROR: [FileService] Ошибка отправки 'куска' для файла '%s': %v", state.FilePath, err)
			state.Status = "failed"
			// TODO: Отправить FileTransferStatus с ошибкой
			return
		}
		offset += int64(bytesRead)
	}

	// 4. Отправляем финальный "кусок" с флагом last_chunk=true
	finalChunk := &protocol.FileData{
		TransferId: state.TransferID,
		LastChunk:  true,
	}
	if err := fs.sendChunk(streamID, finalChunk); err != nil {
		log.Printf("ERROR: [FileService] Ошибка отправки финального 'куска': %v", err)
		state.Status = "failed"
	} else {
		state.Status = "completed"
		log.Printf("INFO: [FileService] Передача файла %s завершена.", state.Metadata.Filename)
	}

	// TODO: Уведомить UI о завершении
}

// sendChunk - helper-функция для упаковки и отправки одного Protobuf-сообщения FileData.
func (fs *FileService) sendChunk(streamID uint64, chunk *protocol.FileData) error {
	// Сериализуем Protobuf-сообщение в байты
	data, err := proto.Marshal(chunk)
	if err != nil {
		return fmt.Errorf("ошибка Marshal 'куска': %w", err)
	}

	// Создаем префикс с длиной сообщения (Varint)
	// Это стандартный способ кадрирования (framing) в libp2p.
	sizePrefix := make([]byte, binary.MaxVarintLen64)
	bytesWritten := binary.PutUvarint(sizePrefix, uint64(len(data)))

	// Отправляем сначала длину...
	if err := fs.core.WriteToStream(streamID, sizePrefix[:bytesWritten]); err != nil {
		return fmt.Errorf("ошибка отправки префикса длины: %w", err)
	}

	// ...потом само сообщение.
	if err := fs.core.WriteToStream(streamID, data); err != nil {
		return fmt.Errorf("ошибка отправки данных 'куска': %w", err)
	}
	return nil
}

// HandleIncomingStream - ИЗМЕНЕНО: теперь он создает Pipe и запускает "читателя".
func (fs *FileService) HandleIncomingStream(payload newcore.NewIncomingStreamPayload) {
	fs.mu.Lock()
	var state *TransferState
	// Ищем активную входящую передачу
	for _, s := range fs.transfers {
		if s.IsIncoming && s.Status == "downloading" { // TODO: Более надежный поиск
			state = s
			break
		}
	}
	if state == nil {
		log.Printf("WARN: [FileService] Получен стрим от %s, но нет активной загрузки.", payload.PeerID)
		fs.core.CloseStream(payload.StreamID)
		fs.mu.Unlock()
		return
	}

	// Создаем "трубу" для этого стрима
	pr, pw := io.Pipe()
	state.pipeReader = pr
	state.pipeWriter = pw
	state.StreamID = payload.StreamID
	fs.mu.Unlock()

	log.Printf("INFO: [FileService] Входящий стрим %d связан с файлом %s. Запуск обработчика.", payload.StreamID, state.Metadata.Filename)
	// Запускаем горутину, которая будет читать из трубы и писать в файл
	go fs.streamFileProcessor(state)
}

		}

		// 4. Закрываем стрим, когда все отправлено
		fs.core.CloseStream(streamID)
		state.Status = "completed"
		log.Printf("INFO: [FileService] Передача файла %s завершена.", state.Metadata.Filename)
	}()
}

// HandleFileAnnouncement (Фаза 2: Получение анонса) - вызывается из Dispatcher'а.
func (fs *FileService) HandleFileAnnouncement(senderID string, metadata *protocol.FileMetadata) (*FileCard, error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()

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
	return fileCard, nil
}

// RequestFileDownload (Фаза 3: Инициация скачивания) - вызывается из UI.

func (fs *FileService) RequestFileDownload(metadata *protocol.FileMetadata, senderID string) {
	log.Printf("INFO: [FileService] Запрашиваем скачивание файла %s от %s", metadata.Filename, senderID[:8])

	// --- ИСПРАВЛЕНИЕ: Сначала обновляем статус, потом отправляем ---
	fs.mu.Lock()
	state, ok := fs.transfers[metadata.TransferId]
	if !ok {
		log.Printf("ERROR: [FileService] Попытка скачать файл с неизвестным transferID: %s", metadata.TransferId)
		fs.mu.Unlock()
		return
	}
	state.Status = "downloading"
	fs.mu.Unlock()

	// Теперь отправляем запрос
	req := &protocol.FileDownloadRequest{
		TransferId: metadata.TransferId,
	}
	chatMsg := &protocol.ChatMessage{
		ChatType: protocol.ChatMessage_PRIVATE,
		ChatId:   senderID,
		Content:  &protocol.ChatMessage_FileRequest{FileRequest: req},
	}
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

	// Обновляем статус у себя
	fs.mu.Lock()
	if state, ok := fs.transfers[metadata.TransferId]; ok {
		state.Status = "downloading"
	}
	fs.mu.Unlock()

	if err := fs.core.SendDataToPeer(senderID, data); err != nil {
		log.Printf("ERROR: [FileService] Не удалось отправить DownloadRequest: %v", err)
	}
}

// HandleStreamData - ИЗМЕНЕНО: теперь он просто пишет в "трубу".
func (fs *FileService) HandleStreamData(payload newcore.StreamDataReceivedPayload) {
	fs.mu.RLock()
	state, ok := fs.findTransferByStreamID(payload.StreamID)
	fs.mu.RUnlock()

	if !ok {
		return
	}

}

// streamFileProcessor - КЛЮЧЕВЫЕ ИЗМЕНЕНИЯ ЗДЕСЬ
func (fs *FileService) streamFileProcessor(state *TransferState) {
	// Отложенное закрытие pipeWriter, чтобы гарантировать выход из цикла чтения
	defer func() {
		if state.pipeWriter != nil {
			state.pipeWriter.Close()
		}
	}()

	homeDir, _ := os.UserHomeDir()
	downloadsPath := filepath.Join(homeDir, "Downloads", "OwlWhisper")
	os.MkdirAll(downloadsPath, 0755)
	filePath := filepath.Join(downloadsPath, state.Metadata.Filename)

	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		log.Printf("ERROR: [FileService] Не удалось создать файл: %v", err)
		return
	}

	// Используем defer, чтобы файл гарантированно закрылся
	defer file.Close()

	streamReader := bufio.NewReader(state.pipeReader)

	for {
		msgLen, err := binary.ReadUvarint(streamReader)
		if err != nil {
			if err != io.EOF {
				log.Printf("ERROR: [FileService] Ошибка чтения длины из pipe: %v", err)
			}
			break
		}

		msgData := make([]byte, msgLen)
		if _, err := io.ReadFull(streamReader, msgData); err != nil {
			log.Printf("ERROR: [FileService] Ошибка чтения 'куска' из pipe: %v", err)
			break
		}

		chunk := &protocol.FileData{}
		if err := proto.Unmarshal(msgData, chunk); err != nil {
			log.Printf("ERROR: [FileService] Ошибка Unmarshal 'куска': %v", err)
			continue
		}

		if chunk.LastChunk {
			log.Printf("INFO: [FileService] Получен последний 'кусок' для %s. Завершение.", state.Metadata.Filename)
			// Мы получили сигнал о конце. Выходим из цикла чтения.
			// Проверка хеша будет после выхода из цикла.
			break
		}

		if _, err := file.Write(chunk.ChunkData); err != nil {
			log.Printf("ERROR: [FileService] Ошибка записи в файл: %v", err)
			break
		}
	}

	// --- ПРОВЕРКА ХЕША ПОСЛЕ ЗАВЕРШЕНИЯ ЦИКЛА ---
	// Важно! file.Close() должен быть вызван до проверки хеша, чтобы
	// все буферы были сброшены на диск. Defer идеально для этого подходит.
	log.Printf("INFO: [FileService] Начинаем проверку хеша для файла %s", state.Metadata.Filename)

	// Переоткрываем файл для чтения
	verifyFile, err := os.Open(filePath)
	if err != nil {
		log.Printf("ERROR: [FileService] Не удалось переоткрыть файл для проверки хеша: %v", err)
		state.Status = "failed"
		return
	}
	defer verifyFile.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, verifyFile); err != nil {
		log.Printf("ERROR: [FileService] Не удалось вычислить хеш: %v", err)
		state.Status = "failed"
		return
	}

	calculatedHash := fmt.Sprintf("%x", hasher.Sum(nil))

	if calculatedHash == state.Metadata.HashSha256 {
		log.Printf("SUCCESS: [FileService] Хеш файла %s совпал!", state.Metadata.Filename)
		state.Status = "completed"
	} else {
		log.Printf("ERROR: [FileService] ХЕШ ФАЙЛА %s НЕ СОВПАЛ!", state.Metadata.Filename)
		log.Printf("  -> Ожидался: %s", state.Metadata.HashSha256)
		log.Printf("  -> Получен:   %s", calculatedHash)
		state.Status = "failed"
	}

	// TODO: Уведомить UI о финальном статусе
}

}

func (fs *FileService) findTransferByStreamID(streamID uint64) (*TransferState, bool) {
	for _, state := range fs.transfers {
		if state.StreamID == streamID {
			return state, true
		}
	}
	return nil, false
}
