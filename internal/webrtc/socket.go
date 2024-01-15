package webrtc

import (
	"log"
	"net/http"
	"webserver/internal/helper"
)

func HandleWebSocketConnections(w http.ResponseWriter, r *http.Request) {
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }

	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	socketId := helper.GenerateUniqueId()
	log.Println("Client Connected")

	err = ws.WriteJSON(webSocketResponse{Type: "connection-success", Data: map[string]interface{}{"socketId": socketId}})
	if err != nil {
		log.Println(err)
		return
	}

	for {
		err := ws.ReadJSON(&request)

		if err != nil {
			log.Println(err)
			return
		}

		switch request.Type {
		case "joinChannel":
			channelId := int64(request.Data["channelId"].(float64))
			joinChannel(socketId, channelId, ws)
		case "createWebRtcTransport":
		case "getProducers":
		case "transport-connect":
		case "transport-produce":
		case "transport-recv-connect":
		case "consume":
		case "consumer-resume":
		}
	}

}
