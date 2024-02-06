package webrtc

import (
	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v3"
	"log"
	//"net/http"
	"strconv"
	"sync"
	"webserver/internal/helper"
)

type VoiceChannel struct {
	channelId int64
	peers     map[int64]*Peer
	mu        sync.Mutex
}

type Peer struct {
	writeMessageToWebSocket func(peer *Peer, data webSocketResponse) error
	connectionId            int64
	ws                      *websocket.Conn
	mu                      sync.Mutex
	AudioTrack              *webrtc.TrackLocalStaticRTP
	VideoTrack              *webrtc.TrackLocalStaticRTP
	peerConnection          *webrtc.PeerConnection
	peerDetails             struct {
		name    string
		isAdmin bool
	}
}

var channels = make(map[int64]*VoiceChannel)

func writeMessageToWebSocket(peer *Peer, data webSocketResponse) error {
	log.Print(&peer.mu)

	peer.mu.Lock()
	err := peer.ws.WriteJSON(data)
	peer.mu.Unlock()
	if err != nil {
		log.Print("Error writing message to websocket:", err)
		return err
	}
	return nil
}

func joinChannel(request webSocketRequest, ws *websocket.Conn) error {
	channelId, _ := strconv.ParseInt(request.Data["channelId"].(string), 10, 64)
	socketId, _ := strconv.ParseInt(request.Data["socketId"].(string), 10, 64)

	channel, ok := channels[channelId]
	if !ok {
		channel = &VoiceChannel{
			channelId: channelId,
			peers:     make(map[int64]*Peer),
		}

	} else {
		channel = channels[channelId]
	}
	peerConnection, err := webrtc.NewPeerConnection(webrtc.Configuration{})
	if err != nil {
		return err
	}

	iceCandidateChan := make(chan webrtc.ICECandidateInit)
	peerConnection.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		if candidate != nil {
			iceCandidateChan <- webrtc.ICECandidateInit{
				Candidate:     candidate.ToJSON().Candidate,
				SDPMid:        candidate.ToJSON().SDPMid,
				SDPMLineIndex: candidate.ToJSON().SDPMLineIndex,
			}
		}
	})

	audioCodecs := webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeOpus, ClockRate: 48000, Channels: 2, SDPFmtpLine: "maxaveragebitrate=3072000"}
	audioTrack, err := webrtc.NewTrackLocalStaticRTP(audioCodecs, strconv.Itoa(int(socketId))+"audio-RTP", "audio")
	if err != nil {
		return err
	}

	videoCodecs := webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeVP8, ClockRate: 90000}
	videoTrack, err := webrtc.NewTrackLocalStaticRTP(videoCodecs, strconv.Itoa(int(socketId))+"video-RTP", "video")
	if err != nil {
		return err
	}

	channel.mu.Lock()
	channel.peers[socketId] = &Peer{writeMessageToWebSocket: writeMessageToWebSocket, ws: ws, peerConnection: peerConnection, connectionId: socketId, AudioTrack: audioTrack, VideoTrack: videoTrack}
	channels[channelId] = channel
	channel.mu.Unlock()

	peerConnection.OnTrack(handleOnTrack(channel.peers[socketId], channelId))

	// rethink design pattern

	go func() {
		for {
			select {
			case iceCandidate := <-iceCandidateChan:

				channel.peers[socketId].writeMessageToWebSocket(channel.peers[socketId], webSocketResponse{Type: "ice-candidate", Data: map[string]interface{}{"candidate": iceCandidate}})
			}
		}
	}()

	return nil
}

func processOffer(request webSocketRequest) (webrtc.SessionDescription, error) {
	channelId, _ := strconv.ParseInt(request.Data["channelId"].(string), 10, 64)
	socketId, _ := strconv.ParseInt(request.Data["socketId"].(string), 10, 64)
	var offer webrtc.SessionDescription
	offerSDP := request.Data["offer"].(map[string]interface{})
	offer.SDP = offerSDP["sdp"].(string)
	sdpTypeStr := offerSDP["type"].(string)
	offerType, err := helper.MapStringToSDPType(sdpTypeStr)
	offer.Type = offerType
	peer := channels[channelId].peers[socketId]
	if err != nil {
		//channels[channelId].peers[socketId].writeMessageToWebSocket(channels[channelId].peers[socketId], webSocketError{Status: http.StatusBadRequest, StatusText: "Invalid request, SDP type is invalid"})

		return webrtc.SessionDescription{}, err
	}

	peerConnection := channels[channelId].peers[socketId].peerConnection
	if err := peerConnection.SetRemoteDescription(offer); err != nil {
		return webrtc.SessionDescription{}, err
	}

	audioTrack := peer.AudioTrack
	videoTrack := peer.VideoTrack

	_, err = peerConnection.AddTrack(audioTrack)
	if err != nil {
		return webrtc.SessionDescription{}, err
	}

	_, err = peerConnection.AddTrack(videoTrack)
	if err != nil {
		return webrtc.SessionDescription{}, err
	}

	answer, err := peerConnection.CreateAnswer(&webrtc.AnswerOptions{
		OfferAnswerOptions: webrtc.OfferAnswerOptions{
			VoiceActivityDetection: true,
		},
	})
	if err != nil {
		return webrtc.SessionDescription{}, err
	}

	if err := peerConnection.SetLocalDescription(answer); err != nil {
		return webrtc.SessionDescription{}, err
	}

	return answer, nil
}

func handleICECandidate(request webSocketRequest, ws *websocket.Conn) error {
	channelId, _ := strconv.ParseInt(request.Data["channelId"].(string), 10, 64)
	socketId, _ := strconv.ParseInt(request.Data["socketId"].(string), 10, 64)
	iceCandidateInit := request.Data["candidate"].(map[string]interface{})
	candidate := iceCandidateInit["candidate"].(string)
	sdpMid := iceCandidateInit["sdpMid"].(string)
	sdpMLineIndex := uint16(iceCandidateInit["sdpMLineIndex"].(float64))

	iceCandidate := webrtc.ICECandidateInit{
		Candidate:     candidate,
		SDPMid:        &sdpMid,
		SDPMLineIndex: &sdpMLineIndex,
	}

	channels[channelId].mu.Lock()
	peer := channels[channelId].peers[socketId]
	err := peer.peerConnection.AddICECandidate(iceCandidate)
	channels[channelId].mu.Unlock()

	if err != nil {
		return err
	}
	return nil
}

func handleOnTrack(peer *Peer, channelId int64) func(track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
	return func(track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
		if track.Kind() == webrtc.RTPCodecTypeAudio {
			log.Println("Received audio track:", track.ID())
		} else if track.Kind() == webrtc.RTPCodecTypeVideo {
			log.Println("Received video track:", track.ID())
		}
		go func() {
			for {
				rtpPacket, _, err := track.ReadRTP()
				if err != nil {
					log.Println("Error reading RTP packet:", err)
					break
				}

				// Forward the RTP packet to other peers in the same channel
				for _, otherPeer := range channels[channelId].peers {
					if otherPeer.connectionId != peer.connectionId {
						if track.Kind() == webrtc.RTPCodecTypeAudio {
							err = otherPeer.AudioTrack.WriteRTP(rtpPacket)
						} else if track.Kind() == webrtc.RTPCodecTypeVideo {
							err = otherPeer.VideoTrack.WriteRTP(rtpPacket)
						}

						if err != nil {
							log.Println("Error forwarding RTP packet:", err)
						}
					}
				}
			}
		}()
	}
}
