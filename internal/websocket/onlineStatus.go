package websocket

import (
	"log"
	"time"
	"webserver/internal/config"
)

func setUserOnline(request webSocketRequest) error {
	userId := request.Data["userId"]
	tx, err := config.UseDBPool().DB.Begin()
	if err != nil {
		return err
	}

	defer func() {
		if err := config.UseDBPool().RollbackOrCommit(tx, err == nil); err != nil {
			log.Println("Error rolling back transaction:", err)
		}
	}()

	_, err = tx.Exec("UPDATE users SET online = true WHERE user_id = ?", userId)
	if err != nil {
		return err
	}

	return nil
}

func setUserOffline(userId int64) error {
	tx, err := config.UseDBPool().DB.Begin()
	if err != nil {
		return err
	}

	defer func() {
		if err := config.UseDBPool().RollbackOrCommit(tx, err == nil); err != nil {
			log.Println("Error rolling back transaction:", err)
		}
	}()

	_, err = tx.Exec("UPDATE users SET last_seen = ?, online = false WHERE user_id = ?", time.Now(), userId)
	if err != nil {
		log.Println(err)
		return err
	}

	return nil
}
