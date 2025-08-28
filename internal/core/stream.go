package core

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
)

// StreamHandler обрабатывает входящие стримы
type StreamHandler struct {
	host          host.Host
	protocolID    protocol.ID
	onMessage     func(peer.ID, []byte)
	onStreamOpen  func(peer.ID, network.Stream)
	onStreamClose func(peer.ID)
}

// NewStreamHandler создает новый обработчик стримов
func NewStreamHandler(host host.Host, protocolID string) *StreamHandler {
	handler := &StreamHandler{
		host:       host,
		protocolID: protocol.ID(protocolID),
	}

	// Регистрируем обработчик стримов
	host.SetStreamHandler(handler.protocolID, handler.handleStream)

	return handler
}

// SetMessageCallback устанавливает callback для входящих сообщений
func (sh *StreamHandler) SetMessageCallback(callback func(peer.ID, []byte)) {
	sh.onMessage = callback
}

// SetStreamOpenCallback устанавливает callback для открытия стримов
func (sh *StreamHandler) SetStreamOpenCallback(callback func(peer.ID, network.Stream)) {
	sh.onStreamOpen = callback
}

// SetStreamCloseCallback устанавливает callback для закрытия стримов
func (sh *StreamHandler) SetStreamCloseCallback(callback func(peer.ID)) {
	sh.onStreamClose = callback
}

// handleStream обрабатывает входящий стрим (аналог handleStream из poc.go)
func (sh *StreamHandler) handleStream(stream network.Stream) {
	remotePeer := stream.Conn().RemotePeer()
	Info("📡 Получен новый стрим от %s", remotePeer.ShortString())

	// Уведомляем об открытии стрима
	if sh.onStreamOpen != nil {
		sh.onStreamOpen(remotePeer, stream)
	}

	// Запускаем обработку стрима
	go sh.handleStreamAsync(stream, remotePeer)
}

// handleStreamAsync асинхронно обрабатывает стрим
func (sh *StreamHandler) handleStreamAsync(stream network.Stream, remotePeer peer.ID) {
	defer func() {
		stream.Close()
		if sh.onStreamClose != nil {
			sh.onStreamClose(remotePeer)
		}
	}()

	// Создаем буферы для чтения и записи
	reader := bufio.NewReader(stream)

	// Читаем входящие сообщения
	for {
		str, err := reader.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				Warn("Ошибка чтения от %s: %v", remotePeer.ShortString(), err)
			}
			return
		}

		// Уведомляем о входящем сообщении
		if sh.onMessage != nil {
			sh.onMessage(remotePeer, []byte(str))
		}
	}
}

// ChatSession представляет сессию чата с пиром
type ChatSession struct {
	stream     network.Stream
	remotePeer peer.ID
	host       host.Host
	writer     *bufio.Writer
	done       chan struct{}
}

// NewChatSession создает новую сессию чата
func NewChatSession(stream network.Stream, host host.Host) *ChatSession {
	return &ChatSession{
		stream:     stream,
		remotePeer: stream.Conn().RemotePeer(),
		host:       host,
		writer:     bufio.NewWriter(stream),
		done:       make(chan struct{}),
	}
}

// Send отправляет любые данные в чат
func (cs *ChatSession) Send(data []byte) error {
	select {
	case <-cs.done:
		return fmt.Errorf("сессия чата закрыта")
	default:
		_, err := cs.writer.Write(data)
		if err != nil {
			return fmt.Errorf("ошибка записи: %w", err)
		}

		err = cs.writer.Flush()
		if err != nil {
			return fmt.Errorf("ошибка flush: %w", err)
		}

		return nil
	}
}

// SendMessage отправляет текстовое сообщение в чат (для обратной совместимости)
func (cs *ChatSession) SendMessage(message string) error {
	return cs.Send([]byte(message))
}

// Close закрывает сессию чата
func (cs *ChatSession) Close() {
	select {
	case <-cs.done:
		return
	default:
		close(cs.done)
		cs.stream.Close()
	}
}

// GetRemotePeer возвращает ID удаленного пира
func (cs *ChatSession) GetRemotePeer() peer.ID {
	return cs.remotePeer
}

// IsClosed проверяет, закрыта ли сессия
func (cs *ChatSession) IsClosed() bool {
	select {
	case <-cs.done:
		return true
	default:
		return false
	}
}

// CreateStream создает исходящий стрим к пиру (аналог NewStream из poc.go)
func (sh *StreamHandler) CreateStream(ctx context.Context, peerID peer.ID, timeout time.Duration) (network.Stream, error) {
	// Создаем контекст с таймаутом для создания стрима
	streamCtx, streamCancel := context.WithTimeout(ctx, timeout)
	defer streamCancel()

	// Создаем стрим с протоколом
	stream, err := sh.host.NewStream(streamCtx, peerID, sh.protocolID)
	if err != nil {
		return nil, fmt.Errorf("не удалось создать стрим: %w", err)
	}

	Info("✅ Стрим успешно создан к %s", peerID.ShortString())
	return stream, nil
}

// CreateStreamWithRetry создает стрим с повторными попытками
func (sh *StreamHandler) CreateStreamWithRetry(ctx context.Context, peerID peer.ID, timeout time.Duration, maxRetries int) (network.Stream, error) {
	var lastErr error

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			Info("🔄 Попытка создания стрима #%d к %s", attempt+1, peerID.ShortString())
			time.Sleep(time.Duration(attempt) * time.Second) // Экспоненциальная задержка
		}

		stream, err := sh.CreateStream(ctx, peerID, timeout)
		if err == nil {
			return stream, nil
		}

		lastErr = err
		Warn("❌ Попытка #%d создания стрима к %s не удалась: %v", attempt+1, peerID.ShortString(), err)
	}

	return nil, fmt.Errorf("все попытки создания стрима провалились: %w", lastErr)
}

// GetActiveStreams возвращает активные соединения к пиру
func (sh *StreamHandler) GetActiveStreams(peerID peer.ID) []network.Conn {
	return sh.host.Network().ConnsToPeer(peerID)
}

// CloseStream закрывает стрим
func (sh *StreamHandler) CloseStream(stream network.Stream) {
	stream.Close()
}

// Send отправляет любые данные пиру через новый стрим
func (sh *StreamHandler) Send(peerID peer.ID, data []byte) error {
	// Создаем стрим с дефолтным таймаутом
	stream, err := sh.CreateStream(context.Background(), peerID, 30*time.Second)
	if err != nil {
		return fmt.Errorf("не удалось создать стрим для отправки: %w", err)
	}
	defer stream.Close()

	// Отправляем данные
	_, err = stream.Write(data)
	if err != nil {
		return fmt.Errorf("не удалось отправить данные: %w", err)
	}

	Info("📤 Отправлено %d байт к %s", len(data), peerID.ShortString())
	return nil
}

// SendMessage отправляет текстовое сообщение пиру (для обратной совместимости)
func (sh *StreamHandler) SendMessage(peerID peer.ID, message string) error {
	return sh.Send(peerID, []byte(message))
}
