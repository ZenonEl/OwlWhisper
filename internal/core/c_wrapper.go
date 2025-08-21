package core

/*
#cgo CFLAGS: -I${SRCDIR}
#include "owlwhisper.h"
#include <stdlib.h>
*/
import "C"
import (
	"context"
	"encoding/json"
	"unsafe"

	"time"

	"github.com/libp2p/go-libp2p/core/peer"
)

// Глобальный экземпляр CoreController
var globalController *CoreController

//export StartOwlWhisper
func StartOwlWhisper() C.int {
	var err error
	globalController, err = NewCoreController(context.Background())
	if err != nil {
		return -1
	}

	err = globalController.Start()
	if err != nil {
		return -1
	}

	return 0
}

//export StopOwlWhisper
func StopOwlWhisper() C.int {
	if globalController == nil {
		return -1
	}

	err := globalController.Stop()
	if err != nil {
		return -1
	}

	globalController = nil
	return 0
}

//export SendMessage
func SendMessage(text *C.char) C.int {
	if globalController == nil {
		return -1
	}

	goText := C.GoString(text)

	// Создаем простое сообщение
	message := []byte(goText)

	err := globalController.Broadcast(message)
	if err != nil {
		return -1
	}

	return 0
}

//export SendMessageToPeer
func SendMessageToPeer(peerID, text *C.char) C.int {
	if globalController == nil {
		return -1
	}

	goPeerID := C.GoString(peerID)
	goText := C.GoString(text)

	// Парсим PeerID
	peer, err := peer.Decode(goPeerID)
	if err != nil {
		return -1
	}

	// Отправляем сообщение
	message := []byte(goText)
	err = globalController.Send(peer, message)
	if err != nil {
		return -1
	}

	return 0
}

//export GetMyPeerID
func GetMyPeerID() *C.char {
	if globalController == nil {
		return C.CString("")
	}

	peerID := globalController.GetMyID()
	return C.CString(peerID)
}

//export GetPeers
func GetPeers() *C.char {
	if globalController == nil {
		return C.CString("[]")
	}

	// Получаем всех пиров из всех источников
	peers := globalController.GetPeers()

	// Если пиров нет, пробуем получить из узла напрямую
	if len(peers) == 0 {
		host := globalController.GetHost()
		if host != nil {
			// Получаем пиров из всех протоколов
			peers = host.Network().Peers()

			// Также проверяем mDNS и DHT
			// TODO: Добавить получение пиров из discovery manager
		}
	}

	peerStrings := make([]string, len(peers))

	for i, p := range peers {
		peerStrings[i] = p.String()
	}

	jsonData, _ := json.Marshal(peerStrings)
	return C.CString(string(jsonData))
}

//export GetConnectionStatus
func GetConnectionStatus() *C.char {
	if globalController == nil {
		return C.CString("{}")
	}

	// Получаем всех пиров из всех источников
	peers := globalController.GetPeers()

	// Если пиров нет, пробуем получить из узла напрямую
	if len(peers) == 0 {
		host := globalController.GetHost()
		if host != nil {
			peers = host.Network().Peers()
		}
	}

	status := map[string]interface{}{
		"connected": len(peers) > 0,
		"peers":     len(peers),
		"my_id":     globalController.GetMyID(),
	}

	jsonData, _ := json.Marshal(status)
	return C.CString(string(jsonData))
}

//export GetChatHistory
func GetChatHistory(peerID *C.char) *C.char {
	// TODO: Реализовать получение истории из storage
	// Пока возвращаем заглушку
	history := []map[string]interface{}{
		{"id": "1", "text": "Привет!", "timestamp": "2025-08-20T20:00:00Z"},
	}
	jsonData, _ := json.Marshal(history)
	return C.CString(string(jsonData))
}

//export GetChatHistoryLimit
func GetChatHistoryLimit(peerID *C.char, limit C.int) *C.char {
	// TODO: Реализовать получение истории с лимитом
	// Пока возвращаем заглушку
	history := []map[string]interface{}{
		{"id": "1", "text": "Привет!", "timestamp": "2025-08-20T20:00:00Z"},
	}
	jsonData, _ := json.Marshal(history)
	return C.CString(string(jsonData))
}

//export ConnectToPeer
func ConnectToPeer(peerID *C.char) C.int {
	if globalController == nil {
		return -1
	}

	goPeerID := C.GoString(peerID)

	// Парсим PeerID
	_, err := peer.Decode(goPeerID)
	if err != nil {
		return -1
	}

	// TODO: Реализовать подключение к пиру
	// Пока просто возвращаем успех
	return 0
}

//export FreeString
func FreeString(str *C.char) {
	C.free(unsafe.Pointer(str))
}

//export GetMyProfile
func GetMyProfile() *C.char {
	if globalController == nil {
		return C.CString("{}")
	}

	profile := globalController.GetMyProfile()
	profileData := map[string]interface{}{
		"nickname":      profile.Nickname,
		"discriminator": profile.Discriminator,
		"display_name":  profile.DisplayName,
		"peer_id":       profile.PeerID,
		"last_seen":     profile.LastSeen.Format(time.RFC3339),
		"is_online":     profile.IsOnline,
	}

	jsonData, _ := json.Marshal(profileData)
	return C.CString(string(jsonData))
}

//export UpdateMyProfile
func UpdateMyProfile(nickname *C.char) C.int {
	if globalController == nil {
		return -1
	}

	goNickname := C.GoString(nickname)
	err := globalController.UpdateMyProfile(goNickname)
	if err != nil {
		return -1
	}

	return 0
}

//export GetPeerProfile
func GetPeerProfile(peerID *C.char) *C.char {
	if globalController == nil {
		return C.CString("{}")
	}

	goPeerID := C.GoString(peerID)
	peer, err := peer.Decode(goPeerID)
	if err != nil {
		return C.CString("{}")
	}

	profile := globalController.GetPeerProfile(peer)
	profileData := map[string]interface{}{
		"nickname":      profile.Nickname,
		"discriminator": profile.Discriminator,
		"display_name":  profile.DisplayName,
		"peer_id":       profile.PeerID,
		"last_seen":     profile.LastSeen.Format(time.RFC3339),
		"is_online":     profile.IsOnline,
	}

	jsonData, _ := json.Marshal(profileData)
	return C.CString(string(jsonData))
}
