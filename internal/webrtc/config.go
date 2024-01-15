package webrtc

import (
	"github.com/gorilla/websocket"
	"github.com/jiyeyuran/mediasoup-go"
	"runtime"
)

var request webSocketRequest

var worker *mediasoup.Worker
var channels = make(map[int64]channel)
var peers = make(map[int64]peer)
var transports []transport
var producers []producer
var consumers []consumer
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

var config = configWebRtc{
	listenInfo: listenInfo{
		protocol: "udp",
		ip:       "127.0.0.1",
	},
	listenPort: 3016,
	mediasoup: mediasoupConfig{
		numWorkers: runtime.NumCPU(),
		router: routerConfig{
			mediaCodecs: []mediaCodec{
				{
					kind:      "audio",
					mimeType:  "audio/opus",
					clockRate: 48000,
					channels:  2,
				},
				{
					kind:      "video",
					mimeType:  "video/VP8",
					clockRate: 90000,
					parameters: map[string]interface{}{
						"x-google-start-bitrate": 1000,
					},
				},
			},
		},
		webRtcTransportConfig: webRtcTransportConfig{
			listenInfos: []listenInfo{
				{
					protocol: "udp",
					ip:       "192.168.1.118",
				},
				{
					protocol: "tcp",
					ip:       "192.168.1.118",
				},
			},
			maxIncomeBitrate:                3072000,
			initialAvailableOutgoingBitrate: 1000000,
			enableUdp:                       true,
			enableTcp:                       true,
			preferUdp:                       true,
		},
	},
}

type channel struct {
	router *mediasoup.Router
	peers  []int64
}

type peer struct {
	webSocket   *websocket.Conn
	socketId    int64
	channelId   int64
	transports  []string
	producers   []string
	consumers   []string
	peerDetails struct {
		name    string
		isAdmin bool
	}
}

type transport struct {
	socketId  int64
	transport *mediasoup.Transport
	channelId int
	consumer  bool
}

type producer struct {
	socketId  int64
	producer  *mediasoup.Producer
	channelId int
}

type consumer struct {
	socketId  int64
	consumer  *mediasoup.Consumer
	channelId int
}

type webSocketRequest struct {
	Type string `json:"type"`
	Data map[string]interface{} `json:"data"`
}

type webSocketResponse struct {
	Type string `json:"type"`
	Data map[string]interface{} `json:"data"`

}

type listenInfo struct {
	protocol string
	ip       string
}

type mediaCodec struct {
	kind       string
	mimeType   string
	clockRate  int
	channels   int
	parameters map[string]interface{}
}

type routerConfig struct {
	mediaCodecs []mediaCodec
}

type webRtcTransportConfig struct {
	listenInfos                     []listenInfo
	maxIncomeBitrate                int
	initialAvailableOutgoingBitrate int
	enableUdp                       bool
	enableTcp                       bool
	preferUdp                       bool
}

type mediasoupConfig struct {
	numWorkers            int
	router                routerConfig
	webRtcTransportConfig webRtcTransportConfig
}

type configWebRtc struct {
	listenInfo listenInfo
	listenPort int
	mediasoup  mediasoupConfig
}
