// Путь: cmd/fyne-gui/ui/file_card.go
package ui

import (
	protocol "OwlWhisper/cmd/fyne-gui/new-core/protocol"
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// FileCard - это кастомный виджет для отображения анонса файла.
type FileCard struct {
	widget.BaseWidget
	metadata   *protocol.FileMetadata
	onDownload func(metadata *protocol.FileMetadata)
}

func NewFileCard(metadata *protocol.FileMetadata, onDownload func(*protocol.FileMetadata)) *FileCard {
	card := &FileCard{
		metadata:   metadata,
		onDownload: onDownload,
	}
	card.ExtendBaseWidget(card)
	return card
}

func (c *FileCard) CreateRenderer() fyne.WidgetRenderer {
	filename := widget.NewLabel(c.metadata.Filename)
	filename.TextStyle.Bold = true

	size := float64(c.metadata.SizeBytes) / 1024.0 / 1024.0 // в МБ
	sizeLabel := widget.NewLabel(fmt.Sprintf("%.2f MB", size))

	downloadButton := widget.NewButton("Скачать", func() {
		c.onDownload(c.metadata)
	})

	return widget.NewSimpleRenderer(container.NewVBox(
		filename,
		sizeLabel,
		downloadButton,
	))
}
