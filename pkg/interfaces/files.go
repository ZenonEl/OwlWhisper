package interfaces

import (
	"context"
	"io"
)

// FileService определяет интерфейс для работы с файлами
type FileService interface {
	// SendFile отправляет файл конкретному пиру
	SendFile(ctx context.Context, peerID string, filePath string) error

	// SendFileData отправляет файл как поток данных
	SendFileData(ctx context.Context, peerID string, fileName string, reader io.Reader) error

	// GetFileInfo возвращает информацию о файле
	GetFileInfo(fileID string) (FileInfo, error)

	// DownloadFile скачивает файл
	DownloadFile(ctx context.Context, fileID string, savePath string) error

	// SubscribeToFiles подписывается на входящие файлы
	SubscribeToFiles() <-chan FileTransfer
}

// FileInfo содержит информацию о файле
type FileInfo struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Size     int64  `json:"size"`
	Type     string `json:"type"`
	Checksum string `json:"checksum"`
}

// FileTransfer представляет передачу файла
type FileTransfer struct {
	ID       string `json:"id"`
	FileName string `json:"fileName"`
	Size     int64  `json:"size"`
	Progress int    `json:"progress"` // 0-100%
	Status   string `json:"status"`   // pending, transferring, completed, failed
}
