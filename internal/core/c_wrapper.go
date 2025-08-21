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
	"sync"
	"time"
	"unsafe"

	"github.com/libp2p/go-libp2p/core/peer"
)

// Глобальный экземпляр CoreController
var globalController *CoreController

// Система управления памятью для строк
var (
	stringPoolMutex sync.RWMutex
	stringPool      = make(map[uintptr]*C.char)
)

// allocString безопасно создает C строку и запоминает её для освобождения
func allocString(s string) *C.char {
	cstr := C.CString(s)
	stringPoolMutex.Lock()
	stringPool[uintptr(unsafe.Pointer(cstr))] = cstr
	stringPoolMutex.Unlock()
	return cstr
}

// freeString безопасно освобождает C строку
func freeString(cstr *C.char) {
	if cstr == nil {
		return
	}
	
	ptr := uintptr(unsafe.Pointer(cstr))
	stringPoolMutex.Lock()
	if _, exists := stringPool[ptr]; exists {
		delete(stringPool, ptr)
		stringPoolMutex.Unlock()
		C.free(unsafe.Pointer(cstr))
	} else {
		stringPoolMutex.Unlock()
		// Строка не найдена в пуле - возможно уже освобождена
	}
}

//export StartOwlWhisper
func StartOwlWhisper() C.int {
	// Инициализируем логгер по умолчанию (только в консоль)
	InitGlobalLogger(LogLevelInfo, LogOutputConsole, "")

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
		return allocString("")
	}

	peerID := globalController.GetMyID()
	return allocString(peerID)
}

//export GetPeers
func GetPeers() *C.char {
	if globalController == nil {
		return allocString("[]")
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
	return allocString(string(jsonData))
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
		"connected":    len(peers) > 0,
		"peers":        len(peers),
		"my_peer_id":   globalController.GetMyID(),
	}

	jsonData, _ := json.Marshal(status)
	return allocString(string(jsonData))
}

//export GetChatHistory
func GetChatHistory(peerID *C.char) *C.char {
	// TODO: Реализовать получение истории из storage
	// Пока возвращаем заглушку
	history := []map[string]interface{}{
		{"id": "1", "text": "Привет!", "timestamp": "2025-08-20T20:00:00Z"},
	}
	jsonData, _ := json.Marshal(history)
	return allocString(string(jsonData))
}

//export GetChatHistoryLimit
func GetChatHistoryLimit(peerID *C.char, limit C.int) *C.char {
	// TODO: Реализовать получение истории с лимитом
	// Пока возвращаем заглушку
	history := []map[string]interface{}{
		{"id": "1", "text": "Привет!", "timestamp": "2025-08-20T20:00:00Z"},
	}
	jsonData, _ := json.Marshal(history)
	return allocString(string(jsonData))
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
	freeString(str)
}

//export GetMyProfile
func GetMyProfile() *C.char {
	if globalController == nil {
		return allocString("{}")
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
	return allocString(string(jsonData))
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
		return allocString("{}")
	}

	goPeerID := C.GoString(peerID)
	peerObj, err := peer.Decode(goPeerID)
	if err != nil {
		return allocString("{}")
	}

	profile := globalController.GetPeerProfile(peerObj)
	profileData := map[string]interface{}{
		"nickname":      profile.Nickname,
		"discriminator": profile.Discriminator,
		"display_name":  profile.DisplayName,
		"peer_id":       profile.PeerID,
		"last_seen":     profile.LastSeen.Format(time.RFC3339),
		"is_online":     profile.IsOnline,
	}

	jsonData, _ := json.Marshal(profileData)
	return allocString(string(jsonData))
}

//export SetLogLevel
func SetLogLevel(level C.int) C.int {
	switch level {
	case 0: // SILENT
		InitGlobalLogger(LogLevelSilent, LogOutputNone, "")
	case 1: // ERROR
		InitGlobalLogger(LogLevelError, LogOutputConsole, "")
	case 2: // WARN
		InitGlobalLogger(LogLevelWarn, LogOutputConsole, "")
	case 3: // INFO
		InitGlobalLogger(LogLevelInfo, LogOutputConsole, "")
	case 4: // DEBUG
		InitGlobalLogger(LogLevelDebug, LogOutputConsole, "")
	default:
		return C.int(1) // Ошибка
	}
	return C.int(0) // Успех
}

//export SetLogOutput
func SetLogOutput(output C.int, logDir *C.char) C.int {
	var logDirStr string
	if logDir != nil {
		logDirStr = C.GoString(logDir)
	}

	var outputType LogOutput
	switch output {
	case 0: // NONE
		outputType = LogOutputNone
	case 1: // CONSOLE
		outputType = LogOutputConsole
	case 2: // FILE
		outputType = LogOutputFile
	case 3: // BOTH
		outputType = LogOutputBoth
	default:
		return C.int(1) // Ошибка
	}

	err := InitGlobalLogger(LogLevelInfo, outputType, logDirStr)
	if err != nil {
		return C.int(1) // Ошибка
	}

	return C.int(0) // Успех
}
