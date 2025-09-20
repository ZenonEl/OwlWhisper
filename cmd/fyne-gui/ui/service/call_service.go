// Путь: cmd/fyne-gui/services/call_service.go

package services

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"sync"
	"sync/atomic"
	"time"

	newcore "OwlWhisper/cmd/fyne-gui/new-core"
	protocol "OwlWhisper/cmd/fyne-gui/new-core/protocol"

	"github.com/gen2brain/malgo"
	"github.com/google/uuid"
	"github.com/pion/webrtc/v4"
	"github.com/pion/webrtc/v4/pkg/media"
	"google.golang.org/protobuf/proto"
	opus "gopkg.in/hraban/opus.v2"
)

const (
	sampleRate                   = 48000
	channels                     = 1
	frameDuration                = 20 * time.Millisecond
	frameSize                    = sampleRate * int(frameDuration) / int(time.Second) // 960
	pcmSizeBytes                 = frameSize * channels * 2                           // 1920
	amplificationFactor          = 20.0                                               // Разумный коэффициент усиления
	playbackStartThresholdFrames = 15                                                 // Кол-во фреймов для старта воспроизведения (300 мс)
	jitterBufferCapacityFrames   = 200                                                // Ёмкость jitter-буфера (4 секунды)
)

const (
	CallStateIdle      CallState = "Idle"
	CallStateDialing   CallState = "Dialing"
	CallStateIncoming  CallState = "Incoming"
	CallStateConnected CallState = "Connected"
)

type CallState string

type JitterBuffer struct {
	packets   chan []byte
	frameSize int
	mu        sync.Mutex
}

type IncomingCallData struct {
	SenderID string
	CallID   string
	Offer    *protocol.CallOffer
}

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

	malgoCtx    *malgo.AllocatedContext
	audioDevice *malgo.Device
	localTrack  *webrtc.TrackLocalStaticSample
	opusEncoder *opus.Encoder
	opusDecoder *opus.Decoder

	jitterBuffer         *JitterBuffer
	captureBuffer        *bytes.Buffer
	playbackLinearBuffer *bytes.Buffer
	doneChan             chan struct{}

	onIncomingCall func(senderID, callID string)

	underflowCounter atomic.Uint64
	isPlaybackReady  atomic.Bool

	pendingCandidates map[string][]webrtc.ICECandidateInit
}

func NewCallService(core newcore.ICoreController, cs *ContactService, onIncomingCall func(string, string)) (*CallService, error) {
	m := &webrtc.MediaEngine{}
	if err := m.RegisterCodec(webrtc.RTPCodecParameters{
		RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeOpus, ClockRate: 48000, Channels: 1, SDPFmtpLine: "minptime=10;useinbandfec=1"},
		PayloadType:        111,
	}, webrtc.RTPCodecTypeAudio); err != nil {
		return nil, err
	}

	api := webrtc.NewAPI(webrtc.WithMediaEngine(m))

	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{{URLs: []string{"stun:stun.l.google.com:19302"}}},
	}

	malgoCtx, err := malgo.InitContext(nil, malgo.ContextConfig{}, func(msg string) { log.Printf("MALGO: %s", msg) })
	if err != nil {
		return nil, fmt.Errorf("ошибка инициализации malgo context: %w", err)
	}

	decoder, err := opus.NewDecoder(sampleRate, channels)
	if err != nil {
		malgoCtx.Uninit()
		malgoCtx.Free()
		return nil, fmt.Errorf("ошибка создания Opus-декодера: %w", err)
	}
	encoder, err := opus.NewEncoder(sampleRate, channels, opus.AppVoIP)
	if err != nil {
		malgoCtx.Uninit()
		malgoCtx.Free()
		return nil, fmt.Errorf("ошибка создания Opus-кодера: %w", err)
	}

	return &CallService{
		core:                 core,
		contactService:       cs,
		webrtcAPI:            api,
		webrtcConfig:         config,
		pendingCandidates:    make(map[string][]webrtc.ICECandidateInit),
		currentState:         CallStateIdle,
		onIncomingCall:       onIncomingCall,
		malgoCtx:             malgoCtx,
		opusDecoder:          decoder,
		opusEncoder:          encoder,
		jitterBuffer:         NewJitterBuffer(pcmSizeBytes, jitterBufferCapacityFrames),
		playbackLinearBuffer: new(bytes.Buffer),
		captureBuffer:        new(bytes.Buffer),
	}, nil
}

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

	pc.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c == nil {
			return
		}
		cs.sendICECandidate(recipientID, cs.currentCallID, c)
	})

	// 1. Создаем локальный аудио-трек, в который будем писать PCM-данные
	audioTrack, err := webrtc.NewTrackLocalStaticSample(webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeOpus, ClockRate: 48000, Channels: 1}, "audio", "pion")
	if err != nil {
		pc.Close()
		return err
	}
	cs.localTrack = audioTrack

	// 2. Добавляем трек в PeerConnection ОДИН РАЗ
	if _, err := pc.AddTrack(audioTrack); err != nil {
		pc.Close()
		return err
	}

	// 3. Устанавливаем обработчик для входящего звука
	pc.OnTrack(func(track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
		log.Printf("INFO: [CallService] (Инициатор) Получен удаленный трек! Codec: %s", track.Codec().MimeType)
		go cs.playRemoteTrack(track)
	})

	// 4. Запускаем аудио-устройство ДО создания Offer
	if err := cs.startAudioDevice(); err != nil {
		pc.Close()
		return err
	}

	if _, err := pc.CreateDataChannel("owl-whisper-data", nil); err != nil {
		cs.HangupCall()
		return fmt.Errorf("не удалось создать Data Channel: %w", err)
	}

	//iceGatheringComplete := webrtc.GatheringCompletePromise(pc)
	pc.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		log.Printf("INFO: [CallService] (Инициатор) Состояние PeerConnection: %s", state.String())
		if state == webrtc.PeerConnectionStateConnected {
			cs.stateMutex.Lock()
			cs.currentState = CallStateConnected
			cs.stateMutex.Unlock()
		}
	})

	offer, err := pc.CreateOffer(nil)
	if err != nil {
		cs.HangupCall()
		return err
	}

	if err = pc.SetLocalDescription(offer); err != nil {
		cs.HangupCall()
		return err
	}

	//<-iceGatheringComplete
	//finalOffer := pc.LocalDescription()

	cs.pcMutex.Lock()
	cs.peerConnection = pc
	cs.pcMutex.Unlock()

	cs.stateMutex.Lock()
	cs.currentTargetPeerID = recipientID
	cs.currentCallID = uuid.New().String()
	cs.currentState = CallStateDialing
	cs.stateMutex.Unlock()

	log.Printf("INFO: [CallService] Сбор ICE завершен. Отправляем готовый Offer...")
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
		cs.HangupCall()
		return err
	}

	return cs.core.SendDataToPeer(recipientID, data)
}

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

	pc.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c == nil {
			return
		}
		cs.sendICECandidate(senderID, callID, c)
	})

	// 1. Устанавливаем обработчик для входящего звука
	pc.OnTrack(func(track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
		log.Printf("INFO: [CallService] (Ответчик) Получен удаленный трек! Codec: %s", track.Codec().MimeType)
		go cs.playRemoteTrack(track)
	})

	// 2. Настраиваем наш исходящий трек для отправки PCM
	audioTrack, err := webrtc.NewTrackLocalStaticSample(webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeOpus, ClockRate: 48000, Channels: 1}, "audio", "pion")
	if err != nil {
		pc.Close()
		return err
	}
	cs.localTrack = audioTrack
	if _, err := pc.AddTrack(audioTrack); err != nil {
		pc.Close()
		return err
	}

	// 3. Запускаем аудио-устройство ДО установки Remote Description
	if err := cs.startAudioDevice(); err != nil {
		pc.Close()
		return err
	}

	pc.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		log.Printf("INFO: [CallService] (Ответчик) Состояние PeerConnection: %s", state.String())
	})

	if err = pc.SetRemoteDescription(webrtc.SessionDescription{
		Type: webrtc.SDPTypeOffer,
		SDP:  offer.Sdp,
	}); err != nil {
		cs.HangupCall()
		return err
	}

	cs.pcMutex.Lock()
	cs.peerConnection = pc
	cs.pcMutex.Unlock()

	cs.applyPendingCandidates_unsafe(senderID)

	answer, err := pc.CreateAnswer(nil)
	if err != nil {
		cs.HangupCall()
		return err
	}

	//gatherComplete := webrtc.GatheringCompletePromise(pc)
	if err = pc.SetLocalDescription(answer); err != nil {
		cs.HangupCall()
		return err
	}
	//<-gatherComplete

	//finalAnswer := pc.LocalDescription()

	cs.stateMutex.Lock()
	cs.currentState = CallStateConnected
	cs.stateMutex.Unlock()

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
		cs.HangupCall()
		return err
	}

	return cs.core.SendDataToPeer(senderID, data)
}

func (cs *CallService) HangupCall() error {
	cs.pcMutex.Lock()
	if cs.peerConnection != nil {
		cs.peerConnection.Close()
		cs.peerConnection = nil
	}
	cs.pcMutex.Unlock()

	cs.stopAudioDevice()

	cs.stateMutex.Lock()
	cs.currentState = CallStateIdle
	cs.currentTargetPeerID = ""
	cs.currentCallID = ""
	cs.incomingCall = nil
	cs.stateMutex.Unlock()

	log.Println("INFO: [CallService] Звонок завершен/отклонен.")
	return nil
}

func (cs *CallService) startAudioDevice() error {
	if cs.audioDevice != nil && cs.audioDevice.IsStarted() {
		return nil
	}

	cs.isPlaybackReady.Store(false)
	cs.captureBuffer.Reset()
	cs.jitterBuffer.Reset()
	cs.playbackLinearBuffer.Reset()
	cs.doneChan = make(chan struct{})

	go cs.startPlaybackLoop()

	deviceConfig := malgo.DefaultDeviceConfig(malgo.Duplex)
	deviceConfig.Capture.Format = malgo.FormatS16
	deviceConfig.Capture.Channels = channels
	deviceConfig.Playback.Format = malgo.FormatS16
	deviceConfig.Playback.Channels = channels
	deviceConfig.SampleRate = sampleRate

	callbacks := malgo.DeviceCallbacks{
		Data: func(output, input []byte, framecount uint32) {

			// --- Блок ЗАХВАТА ---
			cs.captureBuffer.Write(input)
			for cs.captureBuffer.Len() >= pcmSizeBytes {
				pcmBytes := make([]byte, pcmSizeBytes)
				_, _ = cs.captureBuffer.Read(pcmBytes)

				pcmInt16 := pcmBytesToInt16(pcmBytes)

				// Здесь можно применять эффекты к pcmInt16
				// applyRobotEffect(pcmInt16, 50, 0.7)

				opusData := make([]byte, 1000)
				n, err := cs.opusEncoder.Encode(pcmInt16, opusData)
				if err != nil {
					log.Printf("WARN: Ошибка кодирования Opus: %v", err)
					continue
				}
				opusData = opusData[:n]

				if cs.localTrack != nil {
					sample := media.Sample{Data: opusData, Duration: frameDuration}
					if err := cs.localTrack.WriteSample(sample); err != nil && err != io.ErrClosedPipe {
						// log.Printf("WARN: Ошибка записи аудио-сэмпла: %v", err)
					}
				}
			}

			// --- Блок ВОСПРОИЗВЕДЕНИЯ ---
			if !cs.isPlaybackReady.Load() {
				for i := range output {
					output[i] = 0
				}
				return
			}

			n, _ := cs.playbackLinearBuffer.Read(output)
			if n < len(output) {
				cs.underflowCounter.Add(1)
				for i := n; i < len(output); i++ {
					output[i] = 0
				}
			}
		},
	}

	dev, err := malgo.InitDevice(cs.malgoCtx.Context, deviceConfig, callbacks)
	if err != nil {
		close(cs.doneChan)
		return fmt.Errorf("ошибка malgo.InitDevice: %w", err)
	}

	if err := dev.Start(); err != nil {
		dev.Uninit()
		close(cs.doneChan)
		return fmt.Errorf("ошибка malgo.Device.Start: %w", err)
	}
	cs.audioDevice = dev
	log.Println("INFO: [malgo] Аудио-устройство инициализировано и запущено.")
	return nil
}

func (cs *CallService) stopAudioDevice() {
	if cs.doneChan != nil {
		close(cs.doneChan)
		cs.doneChan = nil
	}
	if cs.audioDevice != nil {
		cs.audioDevice.Stop()
		cs.audioDevice.Uninit()
		cs.audioDevice = nil
		log.Println("INFO: [malgo] Аудио-устройство остановлено.")
	}

	if cs.jitterBuffer != nil {
		cs.jitterBuffer.Reset()
	}
	if cs.captureBuffer != nil {
		cs.captureBuffer.Reset()
	}
	if cs.playbackLinearBuffer != nil {
		cs.playbackLinearBuffer.Reset()
	}
}

func (cs *CallService) playRemoteTrack(remoteTrack *webrtc.TrackRemote) {
	pcm := make([]int16, frameSize*channels)
	for {
		rtpPacket, _, err := remoteTrack.ReadRTP()
		if err != nil {
			if err == io.EOF {
				log.Printf("INFO: [WebRTC] Удаленный аудио-поток завершен.")
			} else {
				log.Printf("ERROR: [WebRTC] Ошибка чтения удаленного трека: %v", err)
			}
			return
		}

		if _, err := cs.opusDecoder.Decode(rtpPacket.Payload, pcm); err != nil {
			log.Printf("WARN: Ошибка декодирования Opus: %v", err)
			continue
		}

		pcmBytes := int16ToPcmBytes(pcm)

		amplify(pcmBytes, amplificationFactor)

		if _, err := cs.jitterBuffer.Write(pcmBytes); err != nil {
			// log.Printf("WARN: Ошибка записи в Jitter Buffer (возможно переполнение): %v", err)
		}
	}
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

func (cs *CallService) HandleIncomingOffer(senderID string, callID string, offer *protocol.CallOffer) error {
	cs.stateMutex.Lock()
	if cs.currentState != CallStateIdle {
		cs.stateMutex.Unlock()
		return fmt.Errorf("получен Offer, но состояние не Idle (%s)", cs.currentState)
	}
	log.Printf("INFO: [CallService] Получен входящий звонок. Сохраняем Offer и уведомляем UI.")
	cs.incomingCall = &IncomingCallData{SenderID: senderID, CallID: callID, Offer: offer}
	cs.currentState = CallStateIncoming
	cs.stateMutex.Unlock()
	if cs.onIncomingCall != nil {
		go cs.onIncomingCall(senderID, callID)
	}
	return nil
}

func (cs *CallService) HandleIncomingAnswer(senderID string, callID string, answer *protocol.CallAnswer) error {
	cs.pcMutex.RLock()
	pc := cs.peerConnection
	targetPeer := cs.currentTargetPeerID
	cs.pcMutex.RUnlock()
	if pc == nil || targetPeer != senderID {
		return fmt.Errorf("получен Answer, но нет активного звонка с этим пиром")
	}
	err := pc.SetRemoteDescription(webrtc.SessionDescription{Type: webrtc.SDPTypeAnswer, SDP: answer.Sdp})
	if err != nil {
		return err
	}
	cs.pcMutex.Lock()
	cs.applyPendingCandidates_unsafe(senderID)
	cs.pcMutex.Unlock()
	return nil
}

func (cs *CallService) HandleIncomingICECandidate(senderID string, callID string, candidate *protocol.ICECandidate) error {
	cs.pcMutex.Lock()
	defer cs.pcMutex.Unlock()
	candidateInit := webrtc.ICECandidateInit{Candidate: candidate.CandidateJson}
	if cs.peerConnection != nil && cs.peerConnection.RemoteDescription() != nil {
		return cs.peerConnection.AddICECandidate(candidateInit)
	}
	log.Printf("INFO: [CallService] Получен 'ранний' ICE-кандидат от %s. Буферизируем.", senderID[:8])
	cs.pendingCandidates[senderID] = append(cs.pendingCandidates[senderID], candidateInit)
	return nil
}

func (cs *CallService) sendICECandidate(recipientID string, callID string, c *webrtc.ICECandidate) {
	if c == nil {
		return
	}
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

func (cs *CallService) applyPendingCandidates_unsafe(peerID string) {
	if candidates, ok := cs.pendingCandidates[peerID]; ok {
		log.Printf("INFO: [CallService] Найдены %d 'отложенных' кандидатов для %s. Применяем...", len(candidates), peerID[:8])
		for _, c := range candidates {
			if err := cs.peerConnection.AddICECandidate(c); err != nil {
				log.Printf("WARN: [CallService] Не удалось применить отложенный кандидат: %v", err)
			}
		}
		delete(cs.pendingCandidates, peerID)
	}
}

func NewJitterBuffer(frameSize int, capacity int) *JitterBuffer {
	return &JitterBuffer{
		packets:   make(chan []byte, capacity),
		frameSize: frameSize,
	}
}

func (jb *JitterBuffer) Write(p []byte) (int, error) {
	if len(p) != jb.frameSize {
		return 0, fmt.Errorf("неверный размер фрейма: ожидался %d, получен %d", jb.frameSize, len(p))
	}
	select {
	case jb.packets <- p:
		return len(p), nil
	default:
		// Буфер переполнен: удаляем самый старый пакет и добавляем новый
		<-jb.packets
		jb.packets <- p
		return len(p), fmt.Errorf("jitter buffer overflow")
	}
}

func (jb *JitterBuffer) Reset() {
	jb.mu.Lock()
	defer jb.mu.Unlock()
	for len(jb.packets) > 0 {
		<-jb.packets
	}
}

func pcmBytesToInt16(pcmBytes []byte) []int16 {
	pcmInt16 := make([]int16, len(pcmBytes)/2)
	for i := 0; i < len(pcmInt16); i++ {
		pcmInt16[i] = int16(binary.LittleEndian.Uint16(pcmBytes[i*2:]))
	}
	return pcmInt16
}

func int16ToPcmBytes(pcmInt16 []int16) []byte {
	pcmBytes := make([]byte, len(pcmInt16)*2)
	for i, s := range pcmInt16 {
		binary.LittleEndian.PutUint16(pcmBytes[i*2:], uint16(s))
	}
	return pcmBytes
}

func amplify(pcmData []byte, factor float32) {
	if factor == 1.0 {
		return
	}
	for i := 0; i < len(pcmData); i += 2 {
		sample := int16(binary.LittleEndian.Uint16(pcmData[i : i+2]))
		amplifiedSample := float32(sample) * factor
		if amplifiedSample > 32767 {
			amplifiedSample = 32767
		} else if amplifiedSample < -32768 {
			amplifiedSample = -32768
		}
		binary.LittleEndian.PutUint16(pcmData[i:i+2], uint16(int16(amplifiedSample)))
	}
}

func (cs *CallService) startPlaybackLoop() {
	log.Println("INFO: Playback loop started.")
	defer log.Println("INFO: Playback loop stopped.")

	for {
		select {
		case <-cs.doneChan:
			return
		case packet := <-cs.jitterBuffer.packets:
			cs.playbackLinearBuffer.Write(packet)

			if !cs.isPlaybackReady.Load() && cs.playbackLinearBuffer.Len() >= pcmSizeBytes*playbackStartThresholdFrames {
				log.Println("INFO: Linear playback buffer is filled. Starting audio playback.")
				cs.isPlaybackReady.Store(true)
			}
		}
	}
}
