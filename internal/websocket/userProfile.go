package websocket

import (
	"errors"
	"log"
	"net/http"
	"strconv"
	"webserver/internal/config"
)

func setAppearance(request webSocketRequest) error {
	var userId, _ = strconv.ParseInt(request.Data["userId"].(string), 10, 64)
	appearance := request.Data["appearance"].(int8)
	tx, err := config.UseDBPool().DB.Begin()
	if err != nil {
		return err
	}

	defer func() {
		if err := config.UseDBPool().RollbackOrCommit(tx, err == nil); err != nil {
			log.Println("Error rolling back transaction:", err)
		}
	}()

	_, err = tx.Exec("UPDATE users SET appearance = ? WHERE user_id == ?", userId, appearance)
	if err != nil {
		return err
	}

	return nil
}

func setStatus(request webSocketRequest) (error, int) {
	var userId, _ = strconv.ParseInt(request.Data["userId"].(string), 10, 64)
	status := request.Data["status"].(string)
	tx, err := config.UseDBPool().DB.Begin()
	if err != nil {
		return err, http.StatusInternalServerError
	}

	defer func() {
		if err := config.UseDBPool().RollbackOrCommit(tx, err == nil); err != nil {
			log.Println("Error rolling back transaction:", err)
		}
	}()

	if len(status) <= 128 {
		_, err = tx.Exec("UPDATE users SET status = ? WHERE user_id == ?", userId, status)
		if err != nil {
			return err, http.StatusInternalServerError
		}
	} else {
		return errors.New("status message exceeds the amount of 128 characters"), http.StatusBadRequest
	}

	return nil, http.StatusOK
}

func setPronouns(request webSocketRequest) (error, int) {
	var userId, _ = strconv.ParseInt(request.Data["userId"].(string), 10, 64)
	pronouns := request.Data["pronouns"].(string)
	tx, err := config.UseDBPool().DB.Begin()
	if err != nil {
		return err, http.StatusInternalServerError
	}

	defer func() {
		if err := config.UseDBPool().RollbackOrCommit(tx, err == nil); err != nil {
			log.Println("Error rolling back transaction:", err)
		}
	}()
	if len(pronouns) <= 40 {
		_, err = tx.Exec("UPDATE users SET pronouns = ? WHERE user_id == ?", userId, pronouns)
		if err != nil {
			return err, http.StatusInternalServerError
		}
	} else {
		return errors.New("pronouns exceed the number of 40 characters"), http.StatusBadRequest
	}
	return nil, http.StatusOK
}

func getUserProfile(request webSocketRequest) error {
	var userId, _ = strconv.ParseInt(request.Data["userId"].(string), 10, 64)

	tx, err := config.UseDBPool().DB.Begin()
	if err != nil {
		return err
	}

	defer func() {
		if err := config.UseDBPool().RollbackOrCommit(tx, err == nil); err != nil {
			log.Println("Error rolling back transaction:", err)
		}
	}()

	row, err := tx.Query("SELECT * FROM users WHERE user_id = ?", userId)

	defer func() {
		err := row.Close()
		if err != nil {
			log.Println(err)
			return
		}
	}()

	/*if row.Next() {
		var username string
		var pronouns string
		var bio string
		var joinedAt string
		row.Scan()
		if err != nil {
			log.Println(err)
			return err, http.StatusInternalServerError
		}
	}*/

	return nil
}
