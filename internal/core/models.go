package core

import (
	"github.com/libp2p/go-libp2p/core/peer"
)

// RawMessage представляет собой сырое сообщение от сети
type RawMessage struct {
	SenderID peer.ID
	Data     []byte
}
