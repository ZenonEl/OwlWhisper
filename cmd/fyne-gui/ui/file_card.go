// Путь: cmd/fyne-gui/ui/file_card.go
package ui

import (
	"fmt"

	protocol "OwlWhisper/cmd/fyne-gui/new-core/protocol"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// FileCard - это кастомный виджет для отображения анонса файла.
type FileCard struct {
	widget.BaseWidget
	metadata       *protocol.FileMetadata
	onDownload     func(metadata *protocol.FileMetadata)
	downloadButton *widget.Button
}

// NewFileCardWidget создает новый экземпляр FileCard.
func NewFileCardWidget(metadata *protocol.FileMetadata, onDownload func(*protocol.FileMetadata)) *FileCard {
	card := &FileCard{
		metadata:   metadata,
		onDownload: onDownload,
	}
	card.ExtendBaseWidget(card)
	return card
}

// CreateRenderer создает "рендерер" для нашего виджета.

func (c *FileCard) CreateRenderer() fyne.WidgetRenderer {
	filename := widget.NewLabel(c.metadata.Filename)
	filename.TextStyle.Bold = true

	sizeMB := float64(c.metadata.SizeBytes) / 1024.0 / 1024.0
	sizeLabel := widget.NewLabel(fmt.Sprintf("%.2f MB", sizeMB))

	// ИЗМЕНЕНО: Создаем кнопку и сохраняем на нее ссылку
	c.downloadButton = widget.NewButton("Скачать", func() {
		if c.onDownload != nil {
			// Отключаем кнопку, чтобы предотвратить повторные нажатия
			c.downloadButton.Disable()
			c.downloadButton.SetText("Загрузка...")
			c.onDownload(c.metadata)
		}
	})

	content := container.NewVBox(filename, sizeLabel, c.downloadButton)
	return widget.NewSimpleRenderer(content)
}
