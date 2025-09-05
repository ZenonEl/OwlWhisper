package core

/*
#cgo CFLAGS: -I${SRCDIR}
#include "owlwhisper.h"
#include <stdlib.h>
*/
import "C"
import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sync"
	"unsafe"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
)

// Глобальный экземпляр CoreController
var globalController *CoreController

// Глобальный контекст для управления жизненным циклом
var globalCtx context.Context
var globalCancel context.CancelFunc

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

//export StartOwlWhisperWithKey
func StartOwlWhisperWithKey(keyBytes *C.char, keyLength C.int) C.int {
	// Инициализируем логгер по умолчанию (только в консоль)
	InitGlobalLogger(LogLevelInfo, LogOutputConsole, "")

	// Создаем контекст
	globalCtx, globalCancel = context.WithCancel(context.Background())

	// Конвертируем C строку в Go байты
	goKeyBytes := C.GoBytes(unsafe.Pointer(keyBytes), keyLength)

	// Создаем Core контроллер с переданным ключом
	controller, err := NewCoreControllerWithKeyBytes(globalCtx, goKeyBytes)
	if err != nil {
		Error("❌ Ошибка создания Core контроллера с ключом: %v", err)
		return C.int(1) // Ошибка
	}

	// Запускаем контроллер
	if err := controller.Start(); err != nil {
		Error("❌ Ошибка запуска Core контроллера: %v", err)
		return C.int(1) // Ошибка
	}

	// Сохраняем глобальный экземпляр
	globalController = controller
	return 0
}

//export GenerateNewKeyPair
func GenerateNewKeyPair() *C.char {
	// Инициализируем логгер по умолчанию (только в консоль)
	InitGlobalLogger(LogLevelInfo, LogOutputConsole, "")

	// Генерируем новую пару ключей Ed25519
	privKey, _, err := crypto.GenerateKeyPairWithReader(crypto.Ed25519, 2048, rand.Reader)
	if err != nil {
		Error("❌ Ошибка генерации ключей: %v", err)
		return nil
	}

	// Сериализуем приватный ключ в libp2p формат
	keyBytes, err := crypto.MarshalPrivateKey(privKey)
	if err != nil {
		Error("❌ Ошибка сериализации ключа: %v", err)
		return nil
	}

	// Получаем PeerID из ключа
	peerID, err := peer.IDFromPrivateKey(privKey)
	if err != nil {
		Error("❌ Ошибка получения PeerID: %v", err)
		return nil
	}

	// Создаем JSON с ключом и PeerID
	keyInfo := map[string]interface{}{
		"private_key": base64.StdEncoding.EncodeToString(keyBytes), // Base64 для JSON
		"peer_id":     peerID.String(),
		"key_type":    "Ed25519",
		"key_length":  len(keyBytes),
	}

	// Сериализуем в JSON
	jsonData, err := json.Marshal(keyInfo)
	if err != nil {
		Error("❌ Ошибка сериализации JSON: %v", err)
		return nil
	}

	Info("🔑 Сгенерирована новая пара ключей для PeerID: %s", peerID.String())

	return allocString(string(jsonData))
}

//export GenerateNewKeyBytes
func GenerateNewKeyBytes() *C.char {
	// Генерируем новую пару ключей Ed25519
	privKey, _, err := crypto.GenerateKeyPairWithReader(crypto.Ed25519, 2048, rand.Reader)
	if err != nil {
		Error("❌ Ошибка генерации ключа: %v", err)
		return nil
	}

	// Сериализуем ключ в libp2p формат (сырые байты)
	keyBytes, err := crypto.MarshalPrivateKey(privKey)
	if err != nil {
		Error("❌ Ошибка сериализации ключа: %v", err)
		return nil
	}

	Info("🔑 Сгенерированы сырые байты ключа длиной %d байт", len(keyBytes))

	// Возвращаем base64-encoded строку для безопасной передачи
	encodedKey := base64.StdEncoding.EncodeToString(keyBytes)
	return allocString(encodedKey)
}

//export StopOwlWhisper
func StopOwlWhisper() C.int {
	if globalController == nil {
		return C.int(1) // Ошибка
	}

	err := globalController.Stop()
	if err != nil {
		Error("❌ Ошибка остановки Core контроллера: %v", err)
		return C.int(1) // Ошибка
	}

	// Отменяем контекст
	if globalCancel != nil {
		globalCancel()
	}

	globalController = nil
	return C.int(0) // Успех
}

// NOTE: Broadcast send function removed. Use Send(peerID, data, len) for 1:1 only.

//export Send
func Send(peerID *C.char, data *C.char, dataLength C.int) C.int {
	if globalController == nil {
		return -1
	}

	goPeerID := C.GoString(peerID)
	goData := C.GoBytes(unsafe.Pointer(data), dataLength)

	// Парсим PeerID
	peer, err := peer.Decode(goPeerID)
	if err != nil {
		return -1
	}

	// Отправляем данные (Go-код сам создаст соединение если нужно)
	err = globalController.Send(peer, goData)
	if err != nil {
		return -1
	}

	return 0
}

// NOTE: Legacy SendDataToPeer removed. Use Send(peerID, data, len).

//export GetMyPeerID
func GetMyPeerID() *C.char {
	if globalController == nil {
		return allocString("")
	}

	peerID := globalController.GetMyID()
	return allocString(peerID)
}

//export GetConnectedPeers
func GetConnectedPeers() *C.char {
	if globalController == nil {
		return allocString("[]")
	}

	// Получаем всех пиров из всех источников
	peers := globalController.GetConnectedPeers()

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

//export GetProtectedPeers
func GetProtectedPeers() *C.char {
	if globalController == nil {
		return allocString("[]")
	}

	peers := globalController.GetProtectedPeers()
	peerStrings := make([]string, len(peers))
	for i, p := range peers {
		peerStrings[i] = p.String()
	}

	jsonData, _ := json.Marshal(peerStrings)
	return allocString(string(jsonData))
}

//export AddProtectedPeer
func AddProtectedPeer(peerID *C.char) C.int {
	if globalController == nil {
		return -1
	}

	peerIDStr := C.GoString(peerID)
	peerObj, err := peer.Decode(peerIDStr)
	if err != nil {
		return -1
	}

	err = globalController.AddProtectedPeer(peerObj)
	if err != nil {
		return -1
	}

	return 0
}

//export RemoveProtectedPeer
func RemoveProtectedPeer(peerID *C.char) C.int {
	if globalController == nil {
		return -1
	}

	peerIDStr := C.GoString(peerID)
	peerObj, err := peer.Decode(peerIDStr)
	if err != nil {
		return -1
	}

	err = globalController.RemoveProtectedPeer(peerObj)
	if err != nil {
		return -1
	}

	return 0
}

//export IsProtectedPeer
func IsProtectedPeer(peerID *C.char) C.int {
	if globalController == nil {
		return 0
	}

	peerIDStr := C.GoString(peerID)
	peerObj, err := peer.Decode(peerIDStr)
	if err != nil {
		return 0
	}

	if globalController.IsProtectedPeer(peerObj) {
		return 1
	}
	return 0
}

//export GetConnectionLimits
func GetConnectionLimits() *C.char {
	if globalController == nil {
		return nil
	}

	limits := globalController.GetConnectionLimits()
	jsonData, _ := json.Marshal(limits)
	return allocString(string(jsonData))
}

//export EnableAutoReconnect
func EnableAutoReconnect() C.int {
	if globalController == nil {
		return -1
	}

	globalController.EnableAutoReconnect()
	return 0
}

//export DisableAutoReconnect
func DisableAutoReconnect() C.int {
	if globalController == nil {
		return -1
	}

	globalController.DisableAutoReconnect()
	return 0
}

//export IsAutoReconnectEnabled
func IsAutoReconnectEnabled() C.int {
	if globalController == nil {
		return 0
	}

	if globalController.IsAutoReconnectEnabled() {
		return 1
	}
	return 0
}

//export GetReconnectAttempts
func GetReconnectAttempts(peerID *C.char) C.int {
	if globalController == nil {
		return -1
	}

	peerIDStr := C.GoString(peerID)
	peerObj, err := peer.Decode(peerIDStr)
	if err != nil {
		return -1
	}

	attempts := globalController.GetReconnectAttempts(peerObj)
	return C.int(attempts)
}

//export GetConnectionStatus
func GetConnectionStatus() *C.char {
	if globalController == nil {
		return C.CString("{}")
	}

	// Получаем подключенных пиров
	peers := globalController.GetConnectedPeers()

	status := map[string]interface{}{
		"connected":  len(peers) > 0,
		"peers":      len(peers),
		"my_peer_id": globalController.GetMyID(),
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

//export FindPeer
func FindPeer(peerID *C.char) *C.char {
	if globalController == nil {
		return allocString("{}")
	}

	goPeerID := C.GoString(peerID)
	peerObj, err := peer.Decode(goPeerID)
	if err != nil {
		return allocString("{}")
	}

	addrInfo, err := globalController.FindPeer(peerObj)
	if err != nil {
		errorData := map[string]interface{}{
			"error": err.Error(),
		}
		jsonData, _ := json.Marshal(errorData)
		return allocString(string(jsonData))
	}

	peerData := map[string]interface{}{
		"id":    addrInfo.ID.String(),
		"addrs": addrInfo.Addrs,
	}
	jsonData, _ := json.Marshal(peerData)
	return allocString(string(jsonData))
}

//export GetNetworkStats
func GetNetworkStats() *C.char {
	if globalController == nil {
		return allocString("{}")
	}

	stats := globalController.GetNetworkStats()
	jsonData, _ := json.Marshal(stats)
	return allocString(string(jsonData))
}

//export GetConnectionQuality
func GetConnectionQuality(peerID *C.char) *C.char {
	if globalController == nil {
		return allocString("{}")
	}

	goPeerID := C.GoString(peerID)
	peerObj, err := peer.Decode(goPeerID)
	if err != nil {
		return allocString("{}")
	}

	quality := globalController.GetConnectionQuality(peerObj)
	jsonData, _ := json.Marshal(quality)
	return allocString(string(jsonData))
}

//export SetLogLevel
func SetLogLevel(level C.int) C.int {
	// Получаем текущий логгер
	currentLogger := GetGlobalLogger()
	var currentOutput LogOutput = LogOutputConsole

	if currentLogger != nil {
		// Сохраняем текущие настройки вывода
		currentOutput = currentLogger.output
	}

	switch level {
	case 0: // SILENT
		InitGlobalLogger(LogLevelSilent, currentOutput, "")
	case 1: // ERROR
		InitGlobalLogger(LogLevelError, currentOutput, "")
	case 2: // WARN
		InitGlobalLogger(LogLevelWarn, currentOutput, "")
	case 3: // INFO
		InitGlobalLogger(LogLevelInfo, currentOutput, "")
	case 4: // DEBUG
		InitGlobalLogger(LogLevelDebug, currentOutput, "")
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

//export SavePeerToCache
func SavePeerToCache(peerIDStr *C.char, addresses *C.char, healthy C.int) C.int {
	if globalController == nil {
		return C.int(-1)
	}

	peerID, err := peer.Decode(C.GoString(peerIDStr))
	if err != nil {
		return C.int(-1)
	}

	// Парсим адреса из JSON строки
	var addrList []string
	if err := json.Unmarshal([]byte(C.GoString(addresses)), &addrList); err != nil {
		return C.int(-1)
	}

	isHealthy := healthy != 0

	if err := globalController.SavePeerToCache(peerID, addrList, isHealthy); err != nil {
		return C.int(-1)
	}

	return C.int(0)
}

//export LoadPeerFromCache
func LoadPeerFromCache(peerIDStr *C.char) *C.char {
	if globalController == nil {
		return nil
	}

	peerID, err := peer.Decode(C.GoString(peerIDStr))
	if err != nil {
		return nil
	}

	cachedPeer, err := globalController.LoadPeerFromCache(peerID)
	if err != nil {
		return nil
	}

	// Сериализуем в JSON
	data, err := json.Marshal(cachedPeer)
	if err != nil {
		return nil
	}

	return allocString(string(data))
}

//export Connect
func Connect(peerIDStr *C.char, addrsStr *C.char) C.int {
	if globalController == nil {
		return C.int(-1)
	}

	peerID, err := peer.Decode(C.GoString(peerIDStr))
	if err != nil {
		return C.int(-1)
	}

	// Парсим адреса из JSON строки
	var addrList []string
	if err := json.Unmarshal([]byte(C.GoString(addrsStr)), &addrList); err != nil {
		return C.int(-1)
	}

	// Создаем AddrInfo
	addrInfo := peer.AddrInfo{
		ID:    peerID,
		Addrs: make([]multiaddr.Multiaddr, 0, len(addrList)),
	}

	for _, addrStr := range addrList {
		if addr, err := multiaddr.NewMultiaddr(addrStr); err == nil {
			addrInfo.Addrs = append(addrInfo.Addrs, addr)
		}
	}

	// Подключаемся
	if err := globalController.Connect(addrInfo); err != nil {
		return C.int(-1)
	}

	return C.int(0)
}

//export SetupAutoRelayWithDHT
func SetupAutoRelayWithDHT() C.int {
	if globalController == nil {
		return C.int(-1)
	}

	// Получаем DHT из discovery manager
	if globalController.discovery == nil {
		return C.int(-1)
	}

	dht := globalController.discovery.GetDHT()
	if dht == nil {
		return C.int(-1)
	}

	// Настраиваем autorelay
	if err := globalController.SetupAutoRelayWithDHT(dht); err != nil {
		return C.int(-1)
	}

	return C.int(0)
}

//export GetAllCachedPeers
func GetAllCachedPeers() *C.char {
	if globalController == nil {
		return nil
	}

	cachedPeers, err := globalController.GetAllCachedPeers()
	if err != nil {
		return nil
	}

	// Сериализуем в JSON
	data, err := json.Marshal(cachedPeers)
	if err != nil {
		return nil
	}

	return allocString(string(data))
}

//export GetHealthyCachedPeers
func GetHealthyCachedPeers() *C.char {
	if globalController == nil {
		return nil
	}

	healthyPeers, err := globalController.GetHealthyCachedPeers()
	if err != nil {
		return nil
	}

	// Сериализуем в JSON
	data, err := json.Marshal(healthyPeers)
	if err != nil {
		return nil
	}

	return allocString(string(data))
}

//export RemovePeerFromCache
func RemovePeerFromCache(peerIDStr *C.char) C.int {
	if globalController == nil {
		return C.int(-1)
	}

	peerID, err := peer.Decode(C.GoString(peerIDStr))
	if err != nil {
		return C.int(-1)
	}

	if err := globalController.RemovePeerFromCache(peerID); err != nil {
		return C.int(-1)
	}

	return C.int(0)
}

//export ClearPeerCache
func ClearPeerCache() C.int {
	if globalController == nil {
		return C.int(-1)
	}

	if err := globalController.ClearPeerCache(); err != nil {
		return C.int(-1)
	}

	return C.int(0)
}

//export SaveDHTRoutingTable
func SaveDHTRoutingTable() C.int {
	if globalController == nil {
		return C.int(-1)
	}

	if err := globalController.LoadDHTRoutingTableFromCache(); err != nil {
		return C.int(-1)
	}

	return C.int(0)
}

//export LoadDHTRoutingTableFromCache
func LoadDHTRoutingTableFromCache() C.int {
	if globalController == nil {
		return C.int(-1)
	}

	if err := globalController.LoadDHTRoutingTableFromCache(); err != nil {
		return C.int(-1)
	}

	return C.int(0)
}

//export GetRoutingTableStats
func GetRoutingTableStats() *C.char {
	if globalController == nil {
		return nil
	}

	stats := globalController.GetRoutingTableStats()

	// Сериализуем в JSON
	data, err := json.Marshal(stats)
	if err != nil {
		return nil
	}

	return allocString(string(data))
}

//export GetDHTRoutingTableSize
func GetDHTRoutingTableSize() C.int {
	if globalController == nil {
		return C.int(0)
	}

	size := globalController.GetDHTRoutingTableSize()
	return C.int(size)
}

//export FindProvidersForContent
func FindProvidersForContent(contentID *C.char) *C.char {
	if contentID == nil {
		return C.CString("")
	}

	contentIDStr := C.GoString(contentID)
	if contentIDStr == "" {
		return C.CString("")
	}

	providers, err := globalController.FindProvidersForContent(contentIDStr)
	if err != nil {
		return C.CString(fmt.Sprintf("ERROR: %v", err))
	}

	// Преобразуем []peer.AddrInfo в JSON строку
	var result []map[string]interface{}
	for _, provider := range providers {
		providerInfo := map[string]any{
			"id":     provider.ID.String(),
			"addrs":  provider.Addrs,
			"health": "healthy",
		}
		result = append(result, providerInfo)
	}

	jsonData, err := json.Marshal(result)
	if err != nil {
		return C.CString(fmt.Sprintf("ERROR: не удалось сериализовать результат: %v", err))
	}

	return C.CString(string(jsonData))
}

//export ProvideContent
func ProvideContent(contentID *C.char) C.int {
	if contentID == nil {
		return C.int(-1)
	}

	contentIDStr := C.GoString(contentID)
	if contentIDStr == "" {
		return C.int(-1)
	}

	err := globalController.ProvideContent(contentIDStr)
	if err != nil {
		return C.int(-1)
	}

	return C.int(0)
}

//export GetNextEvent
func GetNextEvent() *C.char {
	if globalController == nil {
		return nil
	}

	// Блокирующе получаем следующее событие
	eventJSON := globalController.GetNextEvent()
	if eventJSON == "" {
		return nil
	}

	// Возвращаем JSON строку события
	return allocString(eventJSON)
}

//export StartAggressiveDiscovery
func StartAggressiveDiscovery(rendezvous *C.char) C.int {
	if globalController == nil {
		return C.int(-1)
	}

	rendezvousStr := C.GoString(rendezvous)
	if rendezvousStr == "" {
		return C.int(-1)
	}

	globalController.StartAggressiveDiscovery(rendezvousStr)
	return C.int(0)
}

//export StartAggressiveAdvertising
func StartAggressiveAdvertising(rendezvous *C.char) C.int {
	if globalController == nil {
		return C.int(-1)
	}

	rendezvousStr := C.GoString(rendezvous)
	if rendezvousStr == "" {
		return C.int(-1)
	}

	globalController.StartAggressiveAdvertising(rendezvousStr)
	return C.int(0)
}

//export FindPeersOnce
func FindPeersOnce(rendezvous *C.char) *C.char {
	if globalController == nil {
		return nil
	}

	rendezvousStr := C.GoString(rendezvous)
	if rendezvousStr == "" {
		return nil
	}

	peers, err := globalController.FindPeersOnce(rendezvousStr)
	if err != nil {
		return C.CString(fmt.Sprintf("ERROR: %v", err))
	}

	// Преобразуем []peer.AddrInfo в JSON строку
	var result []map[string]interface{}
	for _, peer := range peers {
		peerInfo := map[string]any{
			"id":    peer.ID.String(),
			"addrs": peer.Addrs,
		}
		result = append(result, peerInfo)
	}

	jsonData, err := json.Marshal(result)
	if err != nil {
		return C.CString(fmt.Sprintf("ERROR: не удалось сериализовать результат: %v", err))
	}

	return allocString(string(jsonData))
}

//export AdvertiseOnce
func AdvertiseOnce(rendezvous *C.char) C.int {
	if globalController == nil {
		return C.int(-1)
	}

	rendezvousStr := C.GoString(rendezvous)
	if rendezvousStr == "" {
		return C.int(-1)
	}

	err := globalController.AdvertiseOnce(rendezvousStr)
	if err != nil {
		return C.int(-1)
	}

	return C.int(0)
}

//export StartOwlWhisperWithDefaultConfig
func StartOwlWhisperWithDefaultConfig() C.int {
	if globalController != nil {
		return -1 // уже запущен
	}

	// Инициализируем логгер по умолчанию (только в консоль)
	InitGlobalLogger(LogLevelInfo, LogOutputConsole, "")

	globalCtx, globalCancel = context.WithCancel(context.Background())

	controller, err := NewCoreControllerWithConfig(globalCtx, DefaultNodeConfig())
	if err != nil {
		return -1
	}

	if err := controller.Start(); err != nil {
		return -1
	}

	globalController = controller
	return 0
}

//export StartOwlWhisperWithCustomConfig
func StartOwlWhisperWithCustomConfig(configJSON *C.char) C.int {
	if globalController != nil {
		return -1 // уже запущен
	}

	// Инициализируем логгер по умолчанию (только в консоль)
	InitGlobalLogger(LogLevelInfo, LogOutputConsole, "")

	globalCtx, globalCancel = context.WithCancel(context.Background())

	goConfigJSON := C.GoString(configJSON)

	// Парсим JSON конфигурацию
	var config NodeConfig
	err := json.Unmarshal([]byte(goConfigJSON), &config)
	if err != nil {
		return -1
	}

	controller, err := NewCoreControllerWithConfig(globalCtx, &config)
	if err != nil {
		return -1
	}

	if err := controller.Start(); err != nil {
		return -1
	}

	globalController = controller
	return 0
}

//export StartOwlWhisperWithKeyAndCustomConfig
func StartOwlWhisperWithKeyAndCustomConfig(keyBytes *C.char, keyLength C.int, configJSON *C.char) C.int {
	if globalController != nil {
		return -1 // уже запущен
	}

	// Инициализируем логгер по умолчанию (только в консоль)
	InitGlobalLogger(LogLevelInfo, LogOutputConsole, "")

	globalCtx, globalCancel = context.WithCancel(context.Background())

	goKeyBytes := C.GoBytes(unsafe.Pointer(keyBytes), keyLength)
	goConfigJSON := C.GoString(configJSON)

	var config NodeConfig
	if err := json.Unmarshal([]byte(goConfigJSON), &config); err != nil {
		return -1
	}

	controller, err := NewCoreControllerWithKeyBytesAndConfig(globalCtx, goKeyBytes, &config)
	if err != nil {
		return -1
	}

	if err := controller.Start(); err != nil {
		return -1
	}

	globalController = controller
	return 0
}

//export GetCurrentNodeConfig
func GetCurrentNodeConfig() *C.char {
	if globalController == nil {
		return nil
	}

	// Получаем текущую конфигурацию узла
	node := globalController.GetNode()
	if node == nil {
		return nil
	}

	config := node.GetConfig()
	if config == nil {
		return nil
	}

	// Конвертируем в JSON
	configJSON, err := json.Marshal(config)
	if err != nil {
		return nil
	}

	return C.CString(string(configJSON))
}

//export UpdateNodeConfig
func UpdateNodeConfig(configJSON *C.char) C.int {
	if globalController == nil {
		return -1
	}

	goConfigJSON := C.GoString(configJSON)

	// Парсим JSON конфигурацию
	var config NodeConfig
	err := json.Unmarshal([]byte(goConfigJSON), &config)
	if err != nil {
		return -1
	}

	// Обновляем конфигурацию узла
	node := globalController.GetNode()
	if node == nil {
		return -1
	}

	err = node.UpdateConfig(&config)
	if err != nil {
		return -1
	}

	return 0
}
