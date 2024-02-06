package webrtc

import (
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"strconv"
	"webserver/internal/helper"
)

type webSocketResponse struct {
	Type string                 `json:"type"`
	Data map[string]interface{} `json:"data"`
}

type webSocketRequest struct {
	Type string                 `json:"type"`
	Data map[string]interface{} `json:"data"`
}

type webSocketError struct {
	Status     int16  `json:"status"`
	StatusText string `json:"statusText"`
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func HandleWebSocketConnections(w http.ResponseWriter, r *http.Request) {
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }

	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	socketId := helper.GenerateUniqueId()
	log.Println("Client Connected")

	err = ws.WriteJSON(webSocketResponse{Type: "connection-success", Data: map[string]interface{}{"socketId": strconv.FormatInt(socketId, 10)}})
	if err != nil {
		log.Println(err)
		return
	}

	go handleWebSocket(ws, socketId)
}

func handleWebSocket(ws *websocket.Conn, socketId int64) {
	log.Println("handleWebSocket")
	for {
		var request webSocketRequest
		err := ws.ReadJSON(&request)
		if err != nil {
			log.Println(err, request)
			ws.WriteJSON(webSocketError{Status: http.StatusBadRequest, StatusText: "Invalid request, make sure data is an object"})
			return
		}

		log.Println("REQUEST:", request)

		switch request.Type {
		case "joinChannel":
			err := joinChannel(request, ws)
			if err != nil {
				log.Fatalln(err)
				return
			}
			ws.WriteJSON(webSocketResponse{Type: "joinedChannel"})

		case "offer":
			answer, err := processOffer(request)
			if err != nil {
				log.Fatalln(err)
				return
			}
			ws.WriteJSON(webSocketResponse{Type: "answer", Data: map[string]interface{}{"answer": answer}})
		case "ice-candidate":
			err := handleICECandidate(request, ws)
			if err != nil {
				log.Fatalln("Error adding ICE candidate", err)
				return
			}
		case "disconnect":
			handleDisconnect(request)
		}

	}
}

func handleDisconnect(request webSocketRequest) {
	channelId, _ := strconv.ParseInt(request.Data["channelId"].(string), 10, 64)
	socketId, _ := strconv.ParseInt(request.Data["socketId"].(string), 10, 64)

	channels[channelId].mu.Lock()
	delete(channels[channelId].peers, socketId)
	channels[channelId].mu.Unlock()
}
