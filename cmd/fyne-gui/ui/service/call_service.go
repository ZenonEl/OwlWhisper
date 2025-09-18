// Путь: cmd/fyne-gui/services/call_service.go

package services

import (
	"log"
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

	// Текущее активное соединение
	peerConnection *webrtc.PeerConnection
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
		core:           core,
		contactService: cs,
		webrtcAPI:      api,
		webrtcConfig:   config,
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

	// --- Настройка обработчиков событий PeerConnection ---
	// Этот обработчик будет вызываться Pion'ом, когда он найдет
	// нового ICE-кандидата (сетевой маршрут).
	pc.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c == nil {
			return
		}
		// Мы должны отправить этого кандидата нашему собеседнику.
		// TODO: Реализовать отправку ICECandidate сообщения.
		log.Printf("INFO: [CallService] Найден новый ICE-кандидат: %s", c.ToJSON().Candidate)
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
		CallId:  uuid.New().String(), // Генерируем ID для этого звонка
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
	switch payload := msg.Payload.(type) {
	case *protocol.SignalingMessage_Offer:
		// TODO: HandleIncomingOffer(senderID, payload.Offer)
		log.Printf("INFO: [CallService] Получен CallOffer от %s", senderID)
	case *protocol.SignalingMessage_Answer:
		// TODO: HandleIncomingAnswer(senderID, payload.Answer)
		log.Printf("INFO: [CallService] Получен CallAnswer от %s", senderID)
	case *protocol.SignalingMessage_Candidate:
		// TODO: HandleIncomingICECandidate(senderID, payload.Candidate)
		log.Printf("INFO: [CallService] Получен ICECandidate от %s", senderID)
		// ... и так далее
	default:
		print(payload)
	}
}

// TODO: HandleIncomingOffer(senderID string, offer *protocol.CallOffer)
// TODO: HandleIncomingAnswer(senderID string, answer *protocol.CallAnswer)
// TODO: HandleIncomingICECandidate(senderID string, candidate *protocol.ICECandidate)
