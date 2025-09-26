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

	"fyne.io/fyne/v2"
	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"

	newcore "OwlWhisper/cmd/fyne-gui/new-core"
	protocol "OwlWhisper/cmd/fyne-gui/new-core/protocol"
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
}

// FileService управляет всей логикой передачи файлов.
type FileService struct {
	// --- Зависимости ---
	core            newcore.ICoreController
	protocolService IProtocolService
	identityService IIdentityService
	cardGenerator   FileCardGenerator // Зависимость от интерфейса, а не от реализации

	// --- Внутреннее состояние ---
	transfers map[string]*TransferState // Ключ: TransferID
	mu        sync.RWMutex
}

// NewFileService - конструктор для FileService.
func NewFileService(core newcore.ICoreController, protoSvc IProtocolService, idSvc IIdentityService, cardGen FileCardGenerator) *FileService {
	return &FileService{
		core:            core,
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
	if err := fs.core.SendDataToPeer(recipientID, envelopeBytes); err != nil {
		return nil, err
	}

	return metadata, nil
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
	fs.mu.RLock()
	state, ok := fs.transfers[req.TransferId]
	fs.mu.RUnlock()

	if !ok {
		log.Printf("WARN: [FileService] Получен запрос на скачивание неизвестного transferID: %s", req.TransferId)
		// TODO: Отправить FileTransferStatus{UNAVAILABLE}
		return
	}

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
	go fs.processIncomingStream(state, pr) // Передаем pipeReader
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
	if err != nil { /*...*/
		return
	}

	// TODO: Заменить на реальное шифрование
	ciphertext := fileControlBytes
	nonce := []byte("dummy-nonce-files")
	payloadType := fs.protocolService.GetPayloadType(&protocol.FileControl{})
	author := fs.identityService.GetMyIdentityPublicKeyProto()

	envelopeBytes, err := fs.protocolService.CreateSecureEnvelope(author, payloadType, ciphertext, nonce)
	if err != nil { /*...*/
		return
	}

	if err := fs.core.SendDataToPeer(senderID, envelopeBytes); err != nil {
		log.Printf("ERROR: [FileService] Не удалось отправить DownloadRequest: %v", err)
		state.Status = "failed"
	}
}

// streamFileToPeer открывает стрим и передает по нему файл "кусками".
func (fs *FileService) streamFileToPeer(state *TransferState, recipientID string) {
	streamID, err := fs.core.OpenStream(recipientID, newcore.FILE_PROTOCOL_ID)
	if err != nil { /*...*/
		state.Status = "failed"
		return
	}
	defer fs.core.CloseStream(streamID)

	state.StreamID = streamID
	state.Status = "transferring"

	file, err := os.Open(state.FilePath)
	if err != nil { /*...*/
		state.Status = "failed"
		return
	}
	defer file.Close()

	log.Printf("INFO: [FileService] Начата передача файла %s по стриму %d", state.Metadata.Filename, streamID)
	buffer := make([]byte, 64*1024)
	for {
		bytesRead, err := file.Read(buffer)
		if err == io.EOF {
			break
		}
		if err != nil { /*...*/
			state.Status = "failed"
			return
		}

		chunk := &protocol.FileData{TransferId: state.Metadata.TransferId, ChunkData: buffer[:bytesRead]}
		if err := fs.sendChunk(streamID, chunk); err != nil { /*...*/
			state.Status = "failed"
			return
		}
	}

	finalChunk := &protocol.FileData{TransferId: state.Metadata.TransferId, IsLastChunk: true}
	if err := fs.sendChunk(streamID, finalChunk); err == nil {
		state.Status = "completed"
		log.Printf("INFO: [FileService] Передача файла %s завершена.", state.Metadata.Filename)
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
func (fs *FileService) processIncomingStream(state *TransferState, pipeReader *io.PipeReader) {
	defer pipeReader.Close()

	// ... (создание папки и файла)
	homeDir, _ := os.UserHomeDir()
	downloadsPath := filepath.Join(homeDir, "Downloads", "OwlWhisper")
	os.MkdirAll(downloadsPath, 0755)
	filePath := filepath.Join(downloadsPath, state.Metadata.Filename)
	file, err := os.Create(filePath)
	if err != nil { /*...*/
		return
	}

	streamReader := bufio.NewReader(pipeReader)
	for {
		msgLen, err := binary.ReadUvarint(streamReader)
		if err != nil {
			break
		}
		msgData := make([]byte, msgLen)
		if _, err := io.ReadFull(streamReader, msgData); err != nil {
			break
		}

		chunk := &protocol.FileData{}
		if err := proto.Unmarshal(msgData, chunk); err != nil {
			continue
		}
		if chunk.IsLastChunk {
			break
		}
		if _, err := file.Write(chunk.ChunkData); err != nil {
			break
		}
	}
	file.Close() // Закрываем файл перед проверкой хеша

	// --- Проверка хеша ---
	verifyFile, err := os.Open(filePath)
	if err != nil { /*...*/
		state.Status = "failed"
		return
	}
	defer verifyFile.Close()

	hasher := sha256.New()
	io.Copy(hasher, verifyFile)
	calculatedHash := fmt.Sprintf("%x", hasher.Sum(nil))

	if calculatedHash == state.Metadata.HashSha256 {
		log.Printf("SUCCESS: [FileService] Хеш файла %s совпал!", state.Metadata.Filename)
		state.Status = "completed"
	} else {
		log.Printf("ERROR: [FileService] ХЕШ ФАЙЛА %s НЕ СОВПАЛ!", state.Metadata.Filename)
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
