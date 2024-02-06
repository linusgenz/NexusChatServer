package websocket

import (
	"log"
	"strconv"
	"webserver/internal/config"
	"webserver/internal/helper"
)

func saveMessage(request webSocketRequest) error {
	var userId, _ = strconv.ParseInt(request.Data["userId"].(string), 10, 64)
	message := request.Data["message"].(string)
	messageId := helper.GenerateUniqueId()
	var channelId, _ = strconv.ParseInt(request.Data["channelId"].(string), 10, 64)

	tx, err := config.UseDBPool().DB.Begin()
	if err != nil {
		log.Println(err)
		return err
	}

	defer func() {
		if err := config.UseDBPool().RollbackOrCommit(tx, err == nil); err != nil {
			log.Println("Error rolling back transaction:", err)
			return
		}
	}()

	_, err = tx.Exec("INSERT INTO messages (message_id, channel_id, user_id, message_text) VALUES (?,?,?,?)", messageId, channelId, userId, message)
	if err != nil {
		log.Println("Failed add message into db:", err)
		return err
	}

	return nil
}
