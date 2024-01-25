package webrtc

import (
	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v3"
	"log"
	"net/http"
	"strconv"
	"sync"
	"webserver/internal/helper"
)

type VoiceChannel struct {
	channelId int64
	peers     map[int64]Peer
	mu        sync.Mutex
}

type Peer struct {
	webSocket      *websocket.Conn
	AudioTrack     *webrtc.TrackLocalStaticRTP
	VideoTrack     *webrtc.TrackLocalStaticRTP
	peerConnection *webrtc.PeerConnection
	peerDetails    struct {
		name    string
		isAdmin bool
	}
}

var channels = make(map[int64]*VoiceChannel)

func joinChannel(request webSocketRequest, ws *websocket.Conn) error {
	channelId := int64(request.Data["channelId"].(float64))
	socketId := int64(request.Data["socketId"].(float64))

	channel, ok := channels[channelId]
	if !ok {
		channel = &VoiceChannel{
			channelId: channelId,
			peers:     make(map[int64]Peer),
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
	// bitrate etc maxIncomeBitrate: 3072000,

	videoCodecs := webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeVP8, ClockRate: 90000}
	videoTrack, err := webrtc.NewTrackLocalStaticRTP(videoCodecs, strconv.Itoa(int(socketId))+"video-RTP", "video")
	if err != nil {
		return err
	}

	channel.mu.Lock()
	channel.peers[socketId] = Peer{webSocket: ws, peerConnection: peerConnection, AudioTrack: audioTrack, VideoTrack: videoTrack}
	channel.mu.Unlock()
	channels[channelId] = channel

	peerConnection.OnTrack(func(track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
		if track.Kind() == webrtc.RTPCodecTypeAudio {
			log.Println("Received audio track:", track.ID())
		} else if track.Kind() == webrtc.RTPCodecTypeVideo {
			log.Println("Received video track:", track.ID())
		}

		// Forward incoming tracks to other peers
		handleOnTrack(channel.peers[socketId], channelId, track, receiver)
	})

	go func() {
		for {
			select {
			case iceCandidate := <-iceCandidateChan:
				
				channel.peers[socketId].webSocket.WriteJSON(webSocketResponse{Type: "ice-candidate", Data: map[string]interface{}{"candidate": iceCandidate}})
			}
		}
	}()

	return nil
}

func processOffer(request webSocketRequest, ws *websocket.Conn) (webrtc.SessionDescription, error) {
	channelId := int64(request.Data["channelId"].(float64))
	socketId := int64(request.Data["socketId"].(float64))
	var offer webrtc.SessionDescription
	offerSDP := request.Data["offer"].(map[string]interface{})
	offer.SDP = offerSDP["sdp"].(string)
	sdpTypeStr := offerSDP["type"].(string)
	offerType, err := helper.MapStringToSDPType(sdpTypeStr)
	offer.Type = offerType
	if err != nil {
		channels[channelId].peers[socketId].webSocket.WriteJSON(webSocketError{Status: http.StatusBadRequest, StatusText: "Invalid request, SDP type is invalid"})

		return webrtc.SessionDescription{}, err
	}

	peerConnection := channels[channelId].peers[socketId].peerConnection
	if err := peerConnection.SetRemoteDescription(offer); err != nil {
		return webrtc.SessionDescription{}, err
	}

	audioTrack := channels[channelId].peers[socketId].AudioTrack
	videoTrack := channels[channelId].peers[socketId].VideoTrack

	_, err = peerConnection.AddTrack(audioTrack)
	if err != nil {
		return webrtc.SessionDescription{}, err
	}

	_, err = peerConnection.AddTrack(videoTrack)
	if err != nil {
		return webrtc.SessionDescription{}, err
	}

	answer, err := peerConnection.CreateAnswer(nil)
	if err != nil {
		return webrtc.SessionDescription{}, err
	}

	if err := peerConnection.SetLocalDescription(answer); err != nil {
		return webrtc.SessionDescription{}, err
	}

	return answer, nil
}

func handleICECandidate(request webSocketRequest, ws *websocket.Conn) error {
	socketId := int64(request.Data["socketId"].(float64))
	channelId := int64(request.Data["channelId"].(float64))
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

func handleOnTrack(peer Peer, channelId int64, track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
	go func() {
		for {
			rtpPacket, _, err := track.ReadRTP()
			//log.Println("ATTRIBUTES:", attributes)
			if err != nil {
				log.Println("Error reading RTP packet:", err)
				break
			}

			// Forward the RTP packet to other peers in the same channel
			for _, otherPeer := range channels[channelId].peers {
				if otherPeer != peer {
					if track.Kind() == webrtc.RTPCodecTypeAudio {
						err = otherPeer.AudioTrack.WriteRTP(rtpPacket)
					} else if track.Kind() == webrtc.RTPCodecTypeVideo {
						err = otherPeer.VideoTrack.WriteRTP(rtpPacket) // ERR here when user in channel and other joins or e.g.
					}

					if err != nil {
						log.Println("Error forwarding RTP packet:", err)
					}
				}
			}
		}
	}()
}
