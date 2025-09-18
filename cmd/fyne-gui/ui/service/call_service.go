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

// CallService управляет всей логикой, связанной со звонками.
type CallService struct {
	core           newcore.ICoreController
	contactService *ContactService

	// Конфигурация Pion/webrtc API
	webrtcAPI    *webrtc.API
	webrtcConfig webrtc.Configuration

	pcMutex             sync.RWMutex
	peerConnection      *webrtc.PeerConnection
	currentTargetPeerID string
	currentCallID       string

	pendingCandidates map[string][]webrtc.ICECandidateInit
}

func NewCallService(core newcore.ICoreController, cs *ContactService) (*CallService, error) {
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
	}, nil
}

// InitiateCall (Фаза 1: "Звонок") - вызывается из UI.
func (cs *CallService) InitiateCall(recipientID string) error {
	log.Printf("INFO: [CallService] Инициируем звонок пиру %s", recipientID[:8])

	// 1. Создаем новый PeerConnection
	pc, err := cs.webrtcAPI.NewPeerConnection(cs.webrtcConfig)
	if err != nil {
		return err
	}
	cs.peerConnection = pc
	cs.currentTargetPeerID = recipientID
	cs.currentCallID = uuid.New().String()

	// --- Настройка обработчиков событий PeerConnection ---
	// Этот обработчик будет вызываться Pion'ом, когда он найдет
	// нового ICE-кандидата (сетевой маршрут).
	pc.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c == nil {
			return
		}
		log.Printf("INFO: [CallService] (Инициатор) Найден ICE-кандидат, отправляем...")
		cs.sendICECandidate(cs.currentTargetPeerID, cs.currentCallID, c)
	})

	// Этот обработчик вызывается, когда соединение установлено или разорвано.
	pc.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		log.Printf("INFO: [CallService] Состояние PeerConnection изменилось: %s", state.String())
		if state == webrtc.PeerConnectionStateFailed || state == webrtc.PeerConnectionStateClosed {
			// Звонок завершен, можно закрывать соединение.
			cs.peerConnection.Close()
			cs.peerConnection = nil
		}
	})

	// TODO: Здесь мы должны добавить аудио-трек (из микрофона).
	// pc.AddTransceiverFromKind(webrtc.RTPCodecTypeAudio)

	// 2. Создаем SDP Offer
	offer, err := pc.CreateOffer(nil)
	if err != nil {
		return err
	}

	// 3. Устанавливаем этот Offer как наше локальное описание
	if err = pc.SetLocalDescription(offer); err != nil {
		return err
	}

	// 4. Отправляем Offer собеседнику через Core
	log.Printf("INFO: [CallService] SDP Offer создан. Отправляем его...")

	// --- ЗАПОЛНЯЕМ TODO ---
	offerMsg := &protocol.CallOffer{Sdp: offer.SDP}
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

// HandleIncomingOffer обрабатывает входящий CallOffer.
func (cs *CallService) HandleIncomingOffer(senderID string, callID string, offer *protocol.CallOffer) error {
	log.Printf("INFO: [CallService] Обработка CallOffer от %s", senderID[:8])

	// 1. Создаем новый PeerConnection для ответа
	pc, err := cs.webrtcAPI.NewPeerConnection(cs.webrtcConfig)
	if err != nil {
		return err
	}
	cs.peerConnection = pc

	// --- Настраиваем те же самые обработчики, что и при инициации ---
	pc.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c == nil {
			return
		}

		log.Printf("INFO: [CallService] (Ответчик) Найден ICE-кандидат, отправляем...")
		cs.sendICECandidate(senderID, callID, c)
	})

	pc.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		log.Printf("INFO: [CallService] (Ответчик) Состояние PeerConnection изменилось: %s", state.String())
	})

	// TODO: Здесь мы будем настраивать, что делать с входящими треками (аудио/видео)
	// pc.OnTrack(func(track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) { ... })

	// 2. Устанавливаем полученный Offer как "удаленное" описание
	if err = pc.SetRemoteDescription(webrtc.SessionDescription{
		Type: webrtc.SDPTypeOffer,
		SDP:  offer.Sdp,
	}); err != nil {
		return err
	}

	cs.applyPendingCandidates(senderID)

	// 3. Создаем SDP Answer
	answer, err := pc.CreateAnswer(nil)
	if err != nil {
		return err
	}

	// 4. Устанавливаем этот Answer как наше "локальное" описание
	if err = pc.SetLocalDescription(answer); err != nil {
		return err
	}

	// 5. Отправляем Answer обратно инициатору звонка
	log.Printf("INFO: [CallService] SDP Answer создан. Отправляем его...")
	answerMsg := &protocol.CallAnswer{Sdp: answer.SDP}
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
	cs.applyPendingCandidates(senderID)
	// --- КОНЕЦ НОВОГО БЛОКА ---

	return nil
}

// HandleIncomingICECandidate обрабатывает входящий ICE-кандидат.
func (cs *CallService) HandleIncomingICECandidate(senderID string, callID string, candidate *protocol.ICECandidate) error {
	cs.pcMutex.Lock()
	defer cs.pcMutex.Unlock()

	candidateInit := webrtc.ICECandidateInit{Candidate: candidate.CandidateJson}

	// Если PeerConnection еще не создан или мы еще не установили RemoteDescription,
	// то кандидатов принимать рано.
	if cs.peerConnection == nil || cs.peerConnection.RemoteDescription() == nil {
		log.Printf("INFO: [CallService] Получен 'ранний' ICE-кандидат от %s. Буферизируем.", senderID[:8])
		// Сохраняем кандидата в буфер.
		cs.pendingCandidates[senderID] = append(cs.pendingCandidates[senderID], candidateInit)
		return nil
	}

	// Если PeerConnection уже готов, добавляем кандидата немедленно.
	log.Printf("INFO: [CallService] Получен и немедленно добавлен ICE-кандидат от %s", senderID[:8])
	return cs.peerConnection.AddICECandidate(candidateInit)
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

// applyPendingCandidates проверяет буфер и добавляет "отложенные" кандидаты.
func (cs *CallService) applyPendingCandidates(peerID string) {
	cs.pcMutex.Lock()
	defer cs.pcMutex.Unlock()

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
