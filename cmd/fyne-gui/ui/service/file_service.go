// Путь: cmd/fyne-gui/services/file_service.go
package services

import (
	"bufio"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"

	newcore "OwlWhisper/cmd/fyne-gui/new-core"
	protocol "OwlWhisper/cmd/fyne-gui/new-core/protocol"
)

const (
	// Размер одного "окна" данных, после которого мы ждем подтверждение.
	// 16 МБ - хороший баланс между накладными расходами и отзывчивостью.
	fileTransferWindowSize = 16 * 1024 * 1024
)

// FileCardGenerator определяет интерфейс для UI, который умеет создавать виджеты файлов.
// Это позволяет FileService не зависеть от конкретной реализации (Fyne).
type FileCardGenerator interface {
	NewFileCard(metadata *protocol.FileMetadata, onDownload func(*protocol.FileMetadata)) fyne.CanvasObject
}

// TransferState описывает состояние текущей передачи файла.
type TransferState struct {
	IsIncoming bool   // true, если мы скачиваем; false, если мы отдаем
	FilePath   string // Путь к файлу на диске
	Metadata   *protocol.FileMetadata
	StreamID   uint64
	Status     string // "announced", "downloading", "transferring", "completed", "failed"
	pipeWriter *io.PipeWriter
	ackChan    chan int64
}

// FileService управляет всей логикой передачи файлов.
type FileService struct {
	// --- Зависимости ---
	core            newcore.ICoreController
	sender          IMessageSender
	protocolService IProtocolService
	identityService IIdentityService
	cardGenerator   FileCardGenerator // Зависимость от интерфейса, а не от реализации

	// --- Внутреннее состояние ---
	transfers map[string]*TransferState // Ключ: TransferID
	mu        sync.RWMutex
}

// NewFileService - конструктор для FileService.
func NewFileService(core newcore.ICoreController, sender IMessageSender, protoSvc IProtocolService, idSvc IIdentityService, cardGen FileCardGenerator) *FileService {
	return &FileService{
		core:            core,
		sender:          sender,
		protocolService: protoSvc,
		identityService: idSvc,
		cardGenerator:   cardGen,
		transfers:       make(map[string]*TransferState),
	}
}

// ================================================================= //
//                      ПУБЛИЧНЫЕ МЕТОДЫ (API для UI)                  //
// ================================================================= //

// AnnounceFile вычисляет метаданные файла, отправляет анонс по сети и возвращает
// метаданные для отображения в UI отправителя.
func (fs *FileService) AnnounceFile(recipientID, filePath string) (*protocol.FileMetadata, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return nil, err
	}

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return nil, err
	}
	fileHash := fmt.Sprintf("%x", hasher.Sum(nil))

	metadata := &protocol.FileMetadata{
		TransferId: uuid.New().String(),
		Filename:   stat.Name(),
		SizeBytes:  stat.Size(),
		HashSha256: fileHash,
	}

	state := &TransferState{
		IsIncoming: false, // Мы - отправитель
		FilePath:   filePath,
		Metadata:   metadata,
		Status:     "announced",
	}
	fs.mu.Lock()
	fs.transfers[metadata.TransferId] = state
	fs.mu.Unlock()

	// --- Новая логика отправки ---
	// 1. Создаем ChatContent с анонсом файла
	chatContentBytes, err := fs.protocolService.CreateChatContent_FileMetadata(metadata)
	if err != nil {
		return nil, err
	}

	// 2. "Шифруем" (пока заглушка) и упаковываем в SecureEnvelope
	// TODO: Заменить на реальное шифрование
	ciphertext := chatContentBytes
	nonce := []byte("dummy-nonce-files")
	payloadType := fs.protocolService.GetPayloadType(&protocol.ChatContent{})
	author := fs.identityService.GetMyIdentityPublicKeyProto()

	envelopeBytes, err := fs.protocolService.CreateSecureEnvelope(author, payloadType, ciphertext, nonce)
	if err != nil {
		return nil, err
	}

	log.Printf("INFO: [FileService] Анонсируем файл %s пиру %s", metadata.Filename, recipientID[:8])
	if err := fs.sender.SendSecureEnvelope(recipientID, envelopeBytes); err != nil {
		return nil, err
	}

	return metadata, nil
}

func (fs *FileService) HandleChunkAck(ack *protocol.FileChunkAck) {
	fs.mu.RLock()
	state, ok := fs.transfers[ack.TransferId]
	fs.mu.RUnlock()

	if !ok || state.ackChan == nil {
		return // Это ACK для передачи, которую мы не отслеживаем, или она не ждет ACK.
	}

	// Отправляем полученное смещение в канал, чтобы разблокировать отправителя.
	state.ackChan <- ack.AcknowledgedOffset
}

// ================================================================= //
//               ПУБЛИЧНЫЕ МЕТОДЫ (ОБРАБОТЧИКИ от DISPATCHER)         //
// ================================================================= //

// HandleFileAnnouncement вызывается из Dispatcher'а при получении анонса файла.
// Он создает и возвращает виджет для отображения в чате.
func (fs *FileService) HandleFileAnnouncement(senderID string, metadata *protocol.FileMetadata) (fyne.CanvasObject, error) {
	state := &TransferState{
		IsIncoming: true,
		Metadata:   metadata,
		Status:     "announced",
	}
	fs.mu.Lock()
	fs.transfers[metadata.TransferId] = state
	fs.mu.Unlock()

	log.Printf("INFO: [FileService] Получен анонс файла '%s' от %s.", metadata.Filename, senderID[:8])

	// Создаем виджет FileCard через интерфейс
	fileCard := fs.cardGenerator.NewFileCard(metadata, func(m *protocol.FileMetadata) {
		go fs.requestFileDownload(m, senderID)
	})
	return fileCard, nil
}

// HandleDownloadRequest вызывается из Dispatcher'а при получении запроса на скачивание.
func (fs *FileService) HandleDownloadRequest(req *protocol.FileDownloadRequest, senderID string) {
	fs.mu.Lock() // Используем полную блокировку, так как будем изменять state
	state, ok := fs.transfers[req.TransferId]
	if !ok {
		fs.mu.Unlock()
		log.Printf("WARN: [FileService] Получен запрос на скачивание неизвестного transferID: %s", req.TransferId)
		return
	}

	// 1. Создаем и присваиваем канал ДО запуска горутины.
	// Теперь это изменение гарантированно будет видно всем.
	state.ackChan = make(chan int64)
	fs.mu.Unlock()

	log.Printf("INFO: [FileService] Получен запрос на скачивание файла %s от %s", state.Metadata.Filename, senderID[:8])
	go fs.streamFileToPeer(state, senderID)
}

// ================================================================= //
//                  ОБРАБОТЧИКИ СОБЫТИЙ ИЗ CORE                      //
// ================================================================= //

// HandleIncomingStream связывает новый входящий стрим с активной загрузкой.
func (fs *FileService) HandleIncomingStream(payload newcore.NewIncomingStreamPayload) {
	fs.mu.Lock()
	var state *TransferState
	for _, s := range fs.transfers {
		// Ищем передачу, которая ожидает стрима
		if s.IsIncoming && s.Status == "downloading" && s.StreamID == 0 {
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

	pr, pw := io.Pipe()
	state.pipeWriter = pw
	state.StreamID = payload.StreamID
	fs.mu.Unlock()

	log.Printf("INFO: [FileService] Входящий стрим %d связан с файлом %s.", payload.StreamID, state.Metadata.Filename)
	go fs.processIncomingStream(state, pr, payload.PeerID) // Передаем pipeReader
}

// HandleStreamData просто пишет входящие байты в "трубу", связанную со стримом.
func (fs *FileService) HandleStreamData(payload newcore.StreamDataReceivedPayload) {
	fs.mu.RLock()
	state, ok := fs.findTransferByStreamID(payload.StreamID)
	fs.mu.RUnlock()

	if !ok || state.pipeWriter == nil {
		return
	}

	if _, err := state.pipeWriter.Write(payload.Data); err != nil {
		log.Printf("ERROR: [FileService] Ошибка записи в pipe для стрима %d: %v", payload.StreamID, err)
	}
}

// HandleStreamClosed закрывает "трубу" со стороны записи, сигнализируя об окончании.
func (fs *FileService) HandleStreamClosed(payload newcore.StreamClosedPayload) {
	fs.mu.RLock()
	state, ok := fs.findTransferByStreamID(payload.StreamID)
	fs.mu.RUnlock()

	if ok && state.pipeWriter != nil {
		state.pipeWriter.Close()
	}
	log.Printf("INFO: [FileService] Стрим %d закрыт.", payload.StreamID)
}

// ================================================================= //
//                    ВНУТРЕННИЕ МЕТОДЫ (ЛОГИКА)                     //
// ================================================================= //

// requestFileDownload отправляет по сети запрос на начало скачивания.
func (fs *FileService) requestFileDownload(metadata *protocol.FileMetadata, senderID string) {
	fs.mu.Lock()
	state, ok := fs.transfers[metadata.TransferId]
	if !ok || state.Status != "announced" {
		fs.mu.Unlock()
		log.Printf("WARN: [FileService] Попытка скачать файл (%s), который не в статусе 'announced'.", metadata.Filename)
		return
	}
	state.Status = "downloading"
	fs.mu.Unlock()

	log.Printf("INFO: [FileService] Запрашиваем скачивание файла %s от %s", metadata.Filename, senderID[:8])

	// --- Новая логика отправки ---
	fileControlBytes, err := fs.protocolService.CreateFileControl_DownloadRequest(metadata.TransferId)
	if err != nil {
		return
	}

	// TODO: Заменить на реальное шифрование
	ciphertext := fileControlBytes
	nonce := []byte("dummy-nonce-files")
	payloadType := fs.protocolService.GetPayloadType(&protocol.FileControl{})
	author := fs.identityService.GetMyIdentityPublicKeyProto()

	envelopeBytes, err := fs.protocolService.CreateSecureEnvelope(author, payloadType, ciphertext, nonce)
	if err != nil {
		return
	}

	if err := fs.sender.SendSecureEnvelope(senderID, envelopeBytes); err != nil {
		log.Printf("ERROR: [FileService] Не удалось отправить DownloadRequest: %v", err)
		state.Status = "failed"
	}
}

// streamFileToPeer открывает стрим и передает по нему файл "окнами" с ожиданием подтверждений.
func (fs *FileService) streamFileToPeer(state *TransferState, recipientID string) {
	// Defer для закрытия канала ACK. Он должен быть первым, чтобы выполниться последним.
	defer func() {
		if state.ackChan != nil {
			close(state.ackChan)
			state.ackChan = nil
		}
	}()

	streamID, err := fs.core.OpenStream(recipientID, newcore.FILE_PROTOCOL_ID)
	if err != nil {
		log.Printf("ERROR: [FileService SENDER] Не удалось открыть файловый стрим для '%s': %v", state.Metadata.Filename, err)
		state.Status = "failed"
		return
	}
	defer fs.core.CloseStream(streamID)

	state.StreamID = streamID
	state.Status = "transferring"

	file, err := os.Open(state.FilePath)
	if err != nil {
		log.Printf("ERROR: [FileService SENDER] Не удалось открыть файл '%s' для отправки: %v", state.FilePath, err)
		state.Status = "failed"
		return
	}
	defer file.Close()

	log.Printf("INFO: [FileService SENDER] Начата передача файла %s (размер окна: %d MB)", state.Metadata.Filename, fileTransferWindowSize/1024/1024)

	var totalBytesSent int64 = 0
	buffer := make([]byte, 64*1024)

	// Главный цикл отправки "окон"
	for totalBytesSent < state.Metadata.SizeBytes {
		bytesSentInWindow := 0
		// Внутренний цикл: отправляем одно "окно" данных
		for bytesSentInWindow < fileTransferWindowSize && totalBytesSent < state.Metadata.SizeBytes {
			bytesToRead := len(buffer)
			if int64(bytesSentInWindow+bytesToRead) > fileTransferWindowSize {
				bytesToRead = fileTransferWindowSize - bytesSentInWindow
			}
			if leftInFile := state.Metadata.SizeBytes - totalBytesSent; int64(bytesToRead) > leftInFile {
				bytesToRead = int(leftInFile)
			}

			n, err := file.Read(buffer[:bytesToRead])
			if err == io.EOF {
				break
			}
			if err != nil {
				log.Printf("ERROR: [FileService SENDER] Ошибка чтения файла '%s': %v", state.FilePath, err)
				state.Status = "failed"
				return
			}

			chunk := &protocol.FileData{TransferId: state.Metadata.TransferId, ChunkData: buffer[:n]}
			if err := fs.sendChunk(streamID, chunk); err != nil {
				log.Printf("ERROR: [FileService SENDER] Ошибка отправки 'куска' для '%s': %v", state.Metadata.Filename, err)
				state.Status = "failed"
				return
			}

			bytesSentInWindow += n
			totalBytesSent += int64(n)
		}

		// --- ИЗМЕНЕНИЕ ЗДЕСЬ ---
		// Ждем промежуточный ACK, ТОЛЬКО если мы отправили данные, И файл еще НЕ закончен.
		if bytesSentInWindow > 0 && totalBytesSent < state.Metadata.SizeBytes {
			log.Printf("INFO: [FileService SENDER] Отправлено %d MB, ожидание ACK...", totalBytesSent/1024/1024)
			select {
			case offset, ok := <-state.ackChan:
				if !ok {
					log.Printf("ERROR: [FileService SENDER] Канал ACK был закрыт преждевременно.")
					state.Status = "failed"
					return
				}
				if offset < totalBytesSent {
					log.Printf("ERROR: [FileService SENDER] Получен неверный ACK (ожидалось >= %d, получено %d)", totalBytesSent, offset)
					state.Status = "failed"
					return
				}
				log.Printf("INFO: [FileService SENDER] ACK получен. Продолжаем передачу.")
			case <-time.After(60 * time.Second):
				log.Printf("ERROR: [FileService SENDER] Таймаут ожидания ACK.")
				state.Status = "failed"
				return
			}
		}
	}

	// --- ФИНАЛИЗАЦИЯ ПЕРЕДАЧИ ---

	log.Printf("INFO: [FileService SENDER] Все данные отправлены. Отправка final chunk...")
	finalChunk := &protocol.FileData{TransferId: state.Metadata.TransferId, IsLastChunk: true}
	if err := fs.sendChunk(streamID, finalChunk); err != nil {
		log.Printf("ERROR: [FileService SENDER] Ошибка отправки финального 'куска': %v", err)
		state.Status = "failed"
		return
	}

	log.Printf("INFO: [FileService SENDER] Ожидание финального ACK...")
	select {
	case offset, ok := <-state.ackChan:
		if !ok {
			log.Printf("ERROR: [FileService SENDER] Канал ACK был закрыт преждевременно при ожидании финала.")
			state.Status = "failed"
			return
		}
		if offset < totalBytesSent {
			log.Printf("ERROR: [FileService SENDER] Получен неверный финальный ACK (ожидалось >= %d, получено %d)", totalBytesSent, offset)
			state.Status = "failed"
			return
		}
		state.Status = "completed"
		log.Printf("SUCCESS: [FileService SENDER] Финальный ACK получен! Передача файла %s успешно завершена.", state.Metadata.Filename)

	case <-time.After(60 * time.Second):
		log.Printf("ERROR: [FileService SENDER] Таймаут ожидания финального ACK.")
		state.Status = "failed"
		return
	}
}

// sendChunk упаковывает и отправляет один "кусок" файла с префиксом длины.
func (fs *FileService) sendChunk(streamID uint64, chunk *protocol.FileData) error {
	data, err := proto.Marshal(chunk)
	if err != nil {
		return err
	}

	sizePrefix := make([]byte, binary.MaxVarintLen64)
	bytesWritten := binary.PutUvarint(sizePrefix, uint64(len(data)))

	if err := fs.core.WriteToStream(streamID, sizePrefix[:bytesWritten]); err != nil {
		return err
	}
	return fs.core.WriteToStream(streamID, data)
}

// processIncomingStream читает "куски" из "трубы", пишет их в файл и проверяет хеш.
func (fs *FileService) processIncomingStream(state *TransferState, pipeReader *io.PipeReader, senderID string) {
	defer pipeReader.Close()

	homeDir, _ := os.UserHomeDir()
	downloadsPath := filepath.Join(homeDir, "Downloads", "OwlWhisper")
	os.MkdirAll(downloadsPath, 0755)
	filePath := filepath.Join(downloadsPath, state.Metadata.Filename)

	file, err := os.Create(filePath)
	if err != nil {
		log.Printf("ERROR: [FileService RECEIVER] Не удалось создать файл: %v", err)
		state.Status = "failed"
		return
	}
	var totalBytesReceived int64 = 0
	var bytesSinceLastAck int64 = 0
	streamReader := bufio.NewReader(pipeReader)
	for {
		msgLen, err := binary.ReadUvarint(streamReader)
		if err != nil {
			if err != io.EOF && err != io.ErrClosedPipe {
				log.Printf("ERROR: [FileService RECEIVER] Ошибка чтения длины 'куска': %v", err)
			}
			break
		}
		if msgLen == 0 {
			continue
		}

		msgData := make([]byte, msgLen)
		if _, err := io.ReadFull(streamReader, msgData); err != nil {
			log.Printf("ERROR: [FileService RECEIVER] Ошибка чтения данных 'куска': %v", err)
			break
		}

		chunk := &protocol.FileData{}
		if err := proto.Unmarshal(msgData, chunk); err != nil {
			log.Printf("WARN: [FileService RECEIVER] Ошибка Unmarshal 'куска': %v", err)
			continue
		}
		if chunk.IsLastChunk {
			break
		}
		if _, err := file.Write(chunk.ChunkData); err != nil {
			log.Printf("ERROR: [FileService RECEIVER] Ошибка записи в файл: %v", err)
			break
		}
		totalBytesReceived += int64(len(chunk.ChunkData))
		bytesSinceLastAck += int64(len(chunk.ChunkData))

		// 1. Проверяем, не пора ли отправить ACK
		if bytesSinceLastAck >= fileTransferWindowSize {
			log.Printf("INFO: [FileService RECEIVER] Получено %d MB, отправка ACK...", totalBytesReceived/1024/1024)
			fs.sendAck(senderID, state.Metadata.TransferId, totalBytesReceived)
			bytesSinceLastAck = 0 // Сбрасываем счетчик
		}
	}

	// 2. Отправляем финальный ACK после завершения цикла
	log.Printf("INFO: [FileService RECEIVER] Цикл завершен, отправка финального ACK на %d байт", totalBytesReceived)
	fs.sendAck(senderID, state.Metadata.TransferId, totalBytesReceived)

	// --- ИЗМЕНЕНИЯ ЗДЕСЬ ---
	// 1. Принудительно сбрасываем буферы ОС на диск перед закрытием файла.
	if err := file.Sync(); err != nil {
		log.Printf("ERROR: [FileService RECEIVER] Ошибка file.Sync(): %v", err)
	}
	// 2. Закрываем файл, чтобы гарантировать, что все дескрипторы освобождены.
	if err := file.Close(); err != nil {
		log.Printf("ERROR: [FileService RECEIVER] Ошибка file.Close(): %v", err)
	}

	// --- ПРОВЕРКА ХЕША ПОСЛЕ ЗАВЕРШЕНИЯ ---
	log.Printf("INFO: [FileService RECEIVER] Начинаем проверку хеша для файла %s", state.Metadata.Filename)

	verifyFile, err := os.Open(filePath)
	if err != nil {
		log.Printf("ERROR: [FileService RECEIVER] Не удалось переоткрыть файл для проверки хеша: %v", err)
		state.Status = "failed"
		return
	}
	defer verifyFile.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, verifyFile); err != nil {
		log.Printf("ERROR: [FileService RECEIVER] Не удалось вычислить хеш: %v", err)
		state.Status = "failed"
		return
	}

	calculatedHash := fmt.Sprintf("%x", hasher.Sum(nil))

	if calculatedHash == state.Metadata.HashSha256 {
		log.Printf("SUCCESS: [FileService RECEIVER] Хеш файла %s совпал!", state.Metadata.Filename)
		state.Status = "completed"
	} else {
		log.Printf("ERROR: [FileService RECEIVER] ХЕШ ФАЙЛА %s НЕ СОВПАЛ!", state.Metadata.Filename)
		// 2. Добавляем детальное логирование хешей
		log.Printf("  -> Ожидался: %s", state.Metadata.HashSha256)
		log.Printf("  -> Получен:   %s", calculatedHash)
		state.Status = "failed"
	}
}

// findTransferByStreamID - внутренний хелпер.
func (fs *FileService) findTransferByStreamID(streamID uint64) (*TransferState, bool) {
	for _, state := range fs.transfers {
		if state.StreamID == streamID {
			return state, true
		}
	}
	return nil, false
}

func (fs *FileService) sendAck(recipientID, transferID string, offset int64) {
	ackBytes, err := fs.protocolService.CreateFileControl_ChunkAck(transferID, offset)
	if err != nil {
		return
	}

	// TODO: Заменить на реальное шифрование
	ciphertext := ackBytes
	nonce := []byte("dummy-nonce-ack")
	payloadType := fs.protocolService.GetPayloadType(&protocol.FileControl{})
	author := fs.identityService.GetMyIdentityPublicKeyProto()
	envelopeBytes, err := fs.protocolService.CreateSecureEnvelope(author, payloadType, ciphertext, nonce)
	if err != nil {
		return
	}

	// Отправляем ACK "тихо", в фоне.
	if err := fs.sender.SendSecureEnvelope(recipientID, envelopeBytes); err != nil {
		log.Printf("WARN: [FileService] Не удалось отправить ACK для %s", transferID)
	}
}
