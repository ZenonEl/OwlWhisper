// Путь: cmd/fyne-gui/services/call_service.go

package services

import (
	"fmt"
	"log"
	"sync"
	"time"

	protocol "OwlWhisper/cmd/fyne-gui/new-core/protocol"

	"github.com/google/uuid"
	"github.com/pion/webrtc/v4"
	"google.golang.org/protobuf/proto"

	newcore "OwlWhisper/cmd/fyne-gui/new-core"
)

type CallState string

const (
	CallStateIdle      CallState = "Idle"      // Нет активного звонка
	CallStateDialing   CallState = "Dialing"   // Мы инициировали звонок, ждем ответа
	CallStateIncoming  CallState = "Incoming"  // Нам звонят, ждем решения пользователя
	CallStateConnected CallState = "Connected" // Соединение установлено
)

// IncomingCallData хранит информацию о входящем звонке.
type IncomingCallData struct {
	SenderID string
	CallID   string
	Offer    *protocol.CallOffer
}

// CallService управляет всей логикой, связанной со звонками.
type CallService struct {
	core           newcore.ICoreController
	contactService *ContactService

	webrtcAPI    *webrtc.API
	webrtcConfig webrtc.Configuration

	pcMutex             sync.RWMutex
	peerConnection      *webrtc.PeerConnection
	currentTargetPeerID string
	currentCallID       string

	stateMutex   sync.RWMutex
	currentState CallState
	incomingCall *IncomingCallData

	onIncomingCall func(senderID, callID string)

	pendingCandidates map[string][]webrtc.ICECandidateInit
}

func NewCallService(core newcore.ICoreController, cs *ContactService, onIncomingCall func(string, string)) (*CallService, error) {
	// --- Настройка медиа-движка ---
	// Мы пока не используем никакие медиа, поэтому настройки простые.
	m := &webrtc.MediaEngine{}

	// Создаем API Pion с этим движком
	api := webrtc.NewAPI(webrtc.WithMediaEngine(m))

	// Конфигурация PeerConnection. STUN-серверы Google - хороший старт.
	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302"},
			},
		},
	}

	return &CallService{
		core:              core,
		contactService:    cs,
		webrtcAPI:         api,
		webrtcConfig:      config,
		pendingCandidates: make(map[string][]webrtc.ICECandidateInit),
		currentState:      CallStateIdle,
		onIncomingCall:    onIncomingCall,
	}, nil
}

// InitiateCall - ФИНАЛЬНАЯ ВЕРСИЯ. Добавляет Data Channel.
func (cs *CallService) InitiateCall(recipientID string) error {
	cs.stateMutex.Lock()
	if cs.currentState != CallStateIdle {
		cs.stateMutex.Unlock()
		return fmt.Errorf("нельзя начать новый звонок, текущий статус: %s", cs.currentState)
	}
	cs.stateMutex.Unlock()

	log.Printf("INFO: [CallService] Инициируем звонок пиру %s", recipientID[:8])

	pc, err := cs.webrtcAPI.NewPeerConnection(cs.webrtcConfig)
	if err != nil {
		return err
	}

	// --- КЛЮЧЕВОЕ ИЗМЕНЕНИЕ №1: Добавляем Data Channel ---
	// Это заставляет Pion сгенерировать правильный SDP с ice-ufrag.
	// В будущем мы сможем использовать этот канал для отправки данных во время звонка.
	if _, err := pc.CreateDataChannel("owl-whisper-data", nil); err != nil {
		return fmt.Errorf("не удалось создать Data Channel: %w", err)
	}

	iceGatheringComplete := webrtc.GatheringCompletePromise(pc)

	pc.OnICECandidate(func(c *webrtc.ICECandidate) { /* ... (без изменений) */ })
	pc.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		log.Printf("INFO: [CallService] (Инициатор) Состояние PeerConnection: %s", state.String())
	})

	offer, err := pc.CreateOffer(nil)
	if err != nil {
		return err
	}

	if err = pc.SetLocalDescription(offer); err != nil {
		return err
	}

	<-iceGatheringComplete

	finalOffer := pc.LocalDescription()

	// Сохраняем состояния
	cs.stateMutex.Lock()
	cs.peerConnection = pc
	cs.currentTargetPeerID = recipientID
	cs.currentCallID = uuid.New().String()
	cs.currentState = CallStateDialing
	cs.stateMutex.Unlock()

	// Отправляем готовый Offer
	log.Printf("INFO: [CallService] Сбор ICE завершен. Отправляем готовый Offer...")
	offerMsg := &protocol.CallOffer{Sdp: finalOffer.SDP}
	signalMsg := &protocol.SignalingMessage{
		CallId:  cs.currentCallID,
		Payload: &protocol.SignalingMessage_Offer{Offer: offerMsg},
	}

	envelope := &protocol.Envelope{
		MessageId:     uuid.New().String(),
		SenderId:      cs.core.GetMyPeerID(),
		TimestampUnix: time.Now().Unix(),
		Payload:       &protocol.Envelope_SignalingMessage{SignalingMessage: signalMsg},
	}

	data, err := proto.Marshal(envelope)
	if err != nil {
		return err
	}

	return cs.core.SendDataToPeer(recipientID, data)
}

func (cs *CallService) HandleSignalingMessage(senderID string, msg *protocol.SignalingMessage) {
	callID := msg.CallId

	switch payload := msg.Payload.(type) {
	case *protocol.SignalingMessage_Offer:
		if err := cs.HandleIncomingOffer(senderID, callID, payload.Offer); err != nil {
			log.Printf("ERROR: [CallService] Ошибка обработки Offer: %v", err)
		}

	case *protocol.SignalingMessage_Answer:
		if err := cs.HandleIncomingAnswer(senderID, callID, payload.Answer); err != nil {
			log.Printf("ERROR: [CallService] Ошибка обработки Answer: %v", err)
		}

	case *protocol.SignalingMessage_Candidate:
		if err := cs.HandleIncomingICECandidate(senderID, callID, payload.Candidate); err != nil {
			log.Printf("ERROR: [CallService] Ошибка обработки ICE Candidate: %v", err)
		}
	}
}

// HandleIncomingOffer - ФИНАЛЬНАЯ ВЕРСИЯ.
func (cs *CallService) HandleIncomingOffer(senderID string, callID string, offer *protocol.CallOffer) error {
	cs.stateMutex.Lock()
	if cs.currentState != CallStateIdle {
		cs.stateMutex.Unlock()
		return fmt.Errorf("получен Offer, но состояние не Idle (%s)", cs.currentState)
	}

	log.Printf("INFO: [CallService] Получен входящий звонок. Сохраняем Offer и уведомляем UI.", senderID[:8])
	cs.incomingCall = &IncomingCallData{
		SenderID: senderID,
		CallID:   callID,
		Offer:    offer,
	}
	cs.currentState = CallStateIncoming
	cs.stateMutex.Unlock()

	if cs.onIncomingCall != nil {
		go cs.onIncomingCall(senderID, callID)
	}
	return nil
}

// HandleIncomingAnswer обрабатывает входящий CallAnswer.
func (cs *CallService) HandleIncomingAnswer(senderID string, callID string, answer *protocol.CallAnswer) error {
	if cs.peerConnection == nil || cs.currentTargetPeerID != senderID {
		return fmt.Errorf("получен Answer, но нет активного звонка с этим пиром")
	}

	// Устанавливаем полученный Answer как "удаленное" описание.
	err := cs.peerConnection.SetRemoteDescription(webrtc.SessionDescription{
		Type: webrtc.SDPTypeAnswer,
		SDP:  answer.Sdp,
	})
	if err != nil {
		return err
	}

	// --- НОВЫЙ БЛОК: "Слив" буфера ---
	cs.applyPendingCandidates_unsafe(senderID)
	// --- КОНЕЦ НОВОГО БЛОКА ---

	return nil
}

// HandleIncomingICECandidate - ТЕПЕРЬ ВСЕГДА БУФЕРИЗИРУЕТ, если мы - получатель.
func (cs *CallService) HandleIncomingICECandidate(senderID string, callID string, candidate *protocol.ICECandidate) error {
	cs.pcMutex.Lock()
	defer cs.pcMutex.Unlock()

	candidateInit := webrtc.ICECandidateInit{Candidate: candidate.CandidateJson}

	// Если мы - инициатор, и pc уже создан, добавляем сразу.
	if cs.currentState == CallStateDialing && cs.peerConnection != nil {
		return cs.peerConnection.AddICECandidate(candidateInit)
	}

	// Во всех остальных случаях (особенно, когда нам звонят) - буферизируем.
	log.Printf("INFO: [CallService] Получен 'ранний' ICE-кандидат от %s. Буферизируем.", senderID[:8])
	cs.pendingCandidates[senderID] = append(cs.pendingCandidates[senderID], candidateInit)
	return nil
}

// AcceptCall - ФИНАЛЬНАЯ ВЕРСЯ. Добавляет обработчик OnDataChannel.
func (cs *CallService) AcceptCall() error {
	cs.stateMutex.Lock()
	if cs.currentState != CallStateIncoming || cs.incomingCall == nil {
		cs.stateMutex.Unlock()
		return fmt.Errorf("нет входящего звонка для принятия")
	}
	senderID := cs.incomingCall.SenderID
	callID := cs.incomingCall.CallID
	offer := cs.incomingCall.Offer
	cs.incomingCall = nil
	cs.stateMutex.Unlock()

	pc, err := cs.webrtcAPI.NewPeerConnection(cs.webrtcConfig)
	if err != nil {
		return err
	}

	// --- КЛЮЧЕВОЕ ИЗМЕНЕНИЕ №2: Добавляем обработчик Data Channel ---
	// Мы должны быть готовы принять Data Channel, который инициировал собеседник.
	pc.OnDataChannel(func(dc *webrtc.DataChannel) {
		log.Printf("INFO: [CallService] Получен новый Data Channel '%s' от %s", dc.Label(), senderID[:8])
		// TODO: Настроить обработчики для этого канала (OnOpen, OnMessage)
	})

	// Настраиваем обработчики
	pc.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c == nil {
			return
		}
		cs.sendICECandidate(senderID, callID, c)
	})
	pc.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		log.Printf("INFO: [CallService] (Ответчик) Состояние PeerConnection: %s", state.String())
	})

	// 2. Устанавливаем RemoteDescription. Теперь это безопасно.

	if err = pc.SetRemoteDescription(webrtc.SessionDescription{
		Type: webrtc.SDPTypeOffer,
		SDP:  offer.Sdp,
	}); err != nil {
		return err
	}

	cs.applyPendingCandidates_unsafe(senderID)

	answer, err := pc.CreateAnswer(nil)
	if err != nil {
		return err
	}

	gatherComplete := webrtc.GatheringCompletePromise(pc)
	if err = pc.SetLocalDescription(answer); err != nil {
		return err
	}
	<-gatherComplete

	finalAnswer := pc.LocalDescription()

	cs.pcMutex.Lock()
	cs.peerConnection = pc
	cs.pcMutex.Unlock()

	cs.stateMutex.Lock()
	cs.currentState = CallStateConnected
	cs.stateMutex.Unlock()

	// Отправляем ГОТОВЫЙ Answer
	answerMsg := &protocol.CallAnswer{Sdp: finalAnswer.SDP}
	signalMsg := &protocol.SignalingMessage{
		CallId:  callID,
		Payload: &protocol.SignalingMessage_Answer{Answer: answerMsg},
	}
	envelope := &protocol.Envelope{
		MessageId:     uuid.New().String(),
		SenderId:      cs.core.GetMyPeerID(),
		TimestampUnix: time.Now().Unix(),
		Payload:       &protocol.Envelope_SignalingMessage{SignalingMessage: signalMsg},
	}
	data, err := proto.Marshal(envelope)
	if err != nil {
		return err
	}

	return cs.core.SendDataToPeer(senderID, data)
}

// HangupCall - универсальный метод для завершения/отклонения звонка.
func (cs *CallService) HangupCall() error {
	cs.stateMutex.Lock()
	defer cs.stateMutex.Unlock()

	if cs.peerConnection != nil {
		cs.peerConnection.Close()
		cs.peerConnection = nil
	}

	// TODO: Отправить SignalingMessage с Hangup

	cs.currentState = CallStateIdle
	cs.currentTargetPeerID = ""
	cs.currentCallID = ""

	log.Println("INFO: [CallService] Звонок завершен/отклонен.")
	return nil
}

func (cs *CallService) sendICECandidate(recipientID string, callID string, c *webrtc.ICECandidate) {
	candidateMsg := &protocol.ICECandidate{CandidateJson: c.ToJSON().Candidate}
	signalMsg := &protocol.SignalingMessage{
		CallId:  callID,
		Payload: &protocol.SignalingMessage_Candidate{Candidate: candidateMsg},
	}
	envelope := &protocol.Envelope{
		MessageId:     uuid.New().String(),
		SenderId:      cs.core.GetMyPeerID(),
		TimestampUnix: time.Now().Unix(),
		Payload:       &protocol.Envelope_SignalingMessage{SignalingMessage: signalMsg},
	}

	data, err := proto.Marshal(envelope)
	if err != nil {
		log.Printf("ERROR: [CallService] Ошибка Marshal при создании ICE Candidate: %v", err)
		return
	}

	if err := cs.core.SendDataToPeer(recipientID, data); err != nil {
		log.Printf("ERROR: [CallService] Не удалось отправить ICE Candidate: %v", err)
	}
}

// applyPendingCandidates_unsafe проверяет буфер и добавляет "отложенные" кандидаты.
func (cs *CallService) applyPendingCandidates_unsafe(peerID string) {
	if candidates, ok := cs.pendingCandidates[peerID]; ok {
		log.Printf("INFO: [CallService] Найдены %d 'отложенных' кандидатов для %s. Применяем...", len(candidates), peerID[:8])
		for _, c := range candidates {
			if err := cs.peerConnection.AddICECandidate(c); err != nil {
				log.Printf("WARN: [CallService] Не удалось применить отложенный кандидат: %v", err)
			}
		}
		// Очищаем буфер для этого пира
		delete(cs.pendingCandidates, peerID)
	}
}
