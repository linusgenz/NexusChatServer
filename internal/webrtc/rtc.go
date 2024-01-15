package webrtc

import (
	"log"
	"github.com/gorilla/websocket"
)

func removeUser(consumers []consumer, producers []producer, transports []transport, socketId int64) ([]consumer, []producer, []transport) {

	var updatedConsumers []consumer
	var updatedProducers []producer
	var updatedTransports []transport

	for _, consumer := range consumers {
		if consumer.socketId == socketId {
			consumer.consumer.Close()

		} else {
			updatedConsumers = append(updatedConsumers, consumer)
		}
	}

	for _, producer := range producers {
		if producer.socketId == socketId {
			producer.producer.Close()

		} else {
			updatedProducers = append(updatedProducers, producer)
		}
	}

	for _, transport := range transports {
		if transport.socketId == socketId {
			transport.transport.Close()

		} else {
			updatedTransports = append(updatedTransports, transport)
		}
	}

	return updatedConsumers, updatedProducers, updatedTransports
}

func disconnect(socketId int64) {
	log.Println("peer disconnected")

	consumers, producers, transports = removeUser(consumers, producers, transports, socketId)

	if user, ok := peers[socketId]; ok {
		channelId := user.channelId
		delete(peers, socketId)

		channels[channelId] = channel{
			router: channels[channelId].router,
			peers:  filterPeers(channels[channelId].peers, socketId),
		}
	} else {
		log.Fatalln("Socket ID not found in peers map")

	}
}

func filterPeers(slice []int64, socketId int64) []int64 {
	var result []int64
	for _, v := range slice {
		if v != socketId {
			result = append(result, v)
		}
	}
	return result
}

func joinChannel(socketId int64, channelId int64, ws *websocket.Conn) {

	peers[socketId] = peer{
		webSocket: ws,
		socketId: socketId,
		channelId: channelId,
		transports: nil,
		producers: nil,
		consumers: nil,
		peerDetails: struct{name string; isAdmin bool}{
			name: "",
			isAdmin: false,
		},
	};
}

func createChannel(channelId int64, socketId int64) {

}