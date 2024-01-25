package api

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"
	"webserver/internal/helper"
	"github.com/gorilla/mux"
	_ "github.com/mattn/go-sqlite3"
)

type serverData struct {
	Id        int       `json:"id"`
	Name      string    `json:"name"`
	Img       string    `json:"img"`
	CreatedAt time.Time `json:"created_at"`
}

type channelData struct {
	Id        int       `json:"id"`
	ServerId  int       `json:"server_id"`
	Type      uint8     `json:"type"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

type createRes struct {
	GeneralChannelId int64 `json:"general_channel_id"`
	ServerId         int64 `json:"server_id"`
}

func List(w http.ResponseWriter, r *http.Request) {
	var data []serverData

	db, err := sql.Open("sqlite3", "../data/data.sqlite")
	if err != nil {
		http.Error(w, "Failed to open database", http.StatusInternalServerError)
		return
	}

	rows, err := db.Query("SELECT * FROM servers")
	if err != nil {
		http.Error(w, "Failed to execute query", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var id int
		var name string
		var img string
		var created_at time.Time
		err = rows.Scan(&id, &name, &img, &created_at)
		if err != nil {
			http.Error(w, "Failed to scan row", http.StatusInternalServerError)
			return
		}
		row := serverData{
			Id:        id,
			Name:      name,
			Img:       img,
			CreatedAt: created_at,
		}
		data = append(data, row)
	}

	res, err := json.Marshal(data)
	if err != nil {
		http.Error(w, "Failed to encode JSON", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(res)
}

func Channels(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["serverid"]
	var data []channelData

	db, err := sql.Open("sqlite3", "../data/data.sqlite")
	if err != nil {
		http.Error(w, "Failed to open database", 500)
		return
	}

	rows, err := db.Query("SELECT * FROM channels WHERE server_id = ?", id)
	if err != nil {
		http.Error(w, "Failed to execute query", 500)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var id int
		var server_id int
		var channel_type uint8
		var name string
		var created_at time.Time
		err = rows.Scan(&id, &server_id, &channel_type, &name, &created_at)
		if err != nil {
			http.Error(w, "Failed to scan row", 500)
			return
		}
		row := channelData{
			Id:        id,
			ServerId:  server_id,
			Type:      channel_type,
			Name:      name,
			CreatedAt: created_at,
		}
		data = append(data, row)
	}

	res, err := json.Marshal(data)
	if err != nil {
		http.Error(w, "Failed to encode JSON", 500)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(res)
}

func Create(w http.ResponseWriter, r *http.Request) {
	wd, _ := os.Getwd()
	err := r.ParseMultipartForm(32 << 20)
	if err != nil {
		http.Error(w, "Unable to parse form", http.StatusBadRequest)
		return
	}
	formData := r.MultipartForm
	var server_id int64 = helper.GenerateUniqueId()
	var name string = formData.Value["name"][0]
	file, _, err := r.FormFile("img")
	var filename string = strconv.Itoa(int(server_id)) + ".jpg"
	var general_channel_id int64 = helper.GenerateUniqueId()
	var general_voice_id int64 = helper.GenerateUniqueId()
	var imgPath string = filepath.Join(wd,"../" , "public", "img", filename)
	var img_cdn_path string = "https://192.168.1.118:3300/public/img/" + filename

	if err != nil {
		log.Println("No file provided")
	} else {
		defer file.Close()
		buf := bytes.NewBuffer(nil)
		if _, err := io.Copy(buf, file); err != nil {
			log.Println("Error copying file:", err)
			http.Error(w, "Internal Server Error", 500)
			return
		}
		os.WriteFile(imgPath, buf.Bytes(), 0644)

	}

	db, err := sql.Open("sqlite3", "../data/data.sqlite")
	if err != nil {
		http.Error(w, "Failed to open database", 500)
		return
	}
	_, err = db.Exec("INSERT INTO servers (server_name, server_id, img) VALUES (?, ?, ?)", name, server_id, img_cdn_path)
	if err != nil {
		log.Println(err)
		http.Error(w, "Failed add server into db", 500)
		return
	} else {
		log.Printf("A new server has been added with ID %v, and the owner has been added. \n", server_id)
	}

	_, err = db.Exec("INSERT INTO channels (server_id, channel_name, type, channel_id) VALUES (?, 'General', 1, ?), (?, 'General', 2, ?)", server_id, general_channel_id, server_id, general_voice_id)
	if err != nil {
		http.Error(w, "Failed add channels into db", 500)
		return
	} else {
		log.Println(`Standard channels 'general' and 'general' (voice) have been created.`)
	}

	var data []createRes
	data = append(data, createRes{ServerId: server_id, GeneralChannelId: general_channel_id})
	res, err := json.Marshal(data)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(res)
}
