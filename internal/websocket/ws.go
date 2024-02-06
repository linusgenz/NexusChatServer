package websocket

import (
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"strconv"
)

type webSocketRequest struct {
	Type string                 `json:"type"`
	Data map[string]interface{} `json:"data"`
}

type webSocketError struct {
	Status     int    `json:"status"`
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

	var request webSocketRequest
	ws.ReadJSON(&request)
	log.Println(request)
	var userId, ok = strconv.ParseInt(request.Data["userId"].(string), 10, 64)
	if ok != nil {
		log.Println("no userId send")
		return
	}
	go handleWebSocket(ws, userId)
}

func handleWebSocket(ws *websocket.Conn, userId int64) {

	defer func() {
		err := setUserOffline(userId)
		log.Println(userId, "now offline")
		if err != nil {
			ws.WriteJSON(webSocketError{Status: http.StatusInternalServerError, StatusText: "Database operation could not be executed"})
			return
		}
	}()

	for {
		var request webSocketRequest
		if err := ws.ReadJSON(&request); err != nil {
			ws.WriteJSON(webSocketError{Status: http.StatusBadRequest, StatusText: "Invalid request, make sure data is an object"})
			return
		}

		log.Println(request)

		switch request.Type {
		case "alive":
			err := setUserOnline(request)
			if err != nil {
				log.Println(err)
				ws.WriteJSON(webSocketError{Status: http.StatusInternalServerError, StatusText: "Error setting websocket status: Database operation could not be executed"})
				return
			}
		case "update-appearance":
			err := setAppearance(request)
			if err != nil {
				log.Println(err)
				ws.WriteJSON(webSocketError{Status: http.StatusInternalServerError, StatusText: "Error changing websocket appearance: Database operation could not be executed"})
				return
			}
		case "update-status":
			err, statusCode := setStatus(request)
			if err != nil {
				if statusCode == http.StatusInternalServerError {
					log.Println(err)
					ws.WriteJSON(webSocketError{Status: statusCode, StatusText: "Error changing websocket status: Database operation could not be executed"})
				} else if statusCode == http.StatusBadRequest {
					log.Println(err)
					ws.WriteJSON(webSocketError{Status: statusCode, StatusText: "Error changing websocket status: Status message exceeds the amount of 128 characters"})
				}
				return
			}
		case "update-pronouns":
			err, statusCode := setPronouns(request)
			if err != nil {
				if err != nil {
					if statusCode == http.StatusInternalServerError {
						log.Println(err)
						ws.WriteJSON(webSocketError{Status: statusCode, StatusText: "Error changing websocket pronouns: Database operation could not be executed"})
					} else if statusCode == http.StatusBadRequest {
						log.Println(err)
						ws.WriteJSON(webSocketError{Status: statusCode, StatusText: "Error changing websocket pronouns: Pronouns exceed the amount of 40 characters"})
					}
					return
				}
				return
			}
		case "onmessage":
			err := saveMessage(request)
			if err != nil {
				ws.WriteJSON(webSocketError{Status: http.StatusInternalServerError, StatusText: "Error processing message: database operation could not be executed"})
				return
			}
		case "user-profile":
			err := getUserProfile(request)
			if err != nil {
				ws.WriteJSON(webSocketError{Status: http.StatusInternalServerError, StatusText: "Error fetching user profile: database operation could not be executed"})
				return
			}
			/*case "bio-update":
				err := setBio(request)
				if err != nil {
					ws.WriteJSON(webSocketError{Status: http.StatusInternalServerError, StatusText: "Error updating bio: database operation could not be executed"})
					return
				}
			case"custom-status-update":
			case"new-channel":
			case"server-settings-update":
			case"websocket-role-update":
			case"new-thread":
			case"role-update":*/
		}

	}
}
