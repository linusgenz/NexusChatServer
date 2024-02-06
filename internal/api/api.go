package api

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gorilla/mux"
	_ "github.com/mattn/go-sqlite3"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"time"
	"webserver/internal/config"
	"webserver/internal/helper"
)

type serverData struct {
	ServerId  string    `json:"serverId"`
	Name      string    `json:"name"`
	Img       string    `json:"img"`
	CreatedAt time.Time `json:"createdAt"`
	UserData  struct {
		JoinedAt     time.Time `json:"createdAt"`
		ServerOwner  bool      `json:"ServerOwner"`
		MembershipId int       `json:"membershipId"`
	} `json:"userData"`
}

type userData struct {
	UserId      string    `json:"userId"`
	Username    string    `json:"serverId"`
	DisplayName string    `json:"displayName"`
	Appearance  uint8     `json:"appearance"`
	Bio         string    `json:"bio"`
	Status      string    `json:"status"`
	JoinedAt    time.Time `json:"joinedAt"`
	LastSeen    time.Time `json:"lastSeen"`
	Pronouns    string    `json:"pronouns"`
	Img         string    `json:"img"`
	Online      bool      `json:"online"`
}

type channelData struct {
	Id        string    `json:"id"`
	ServerId  string    `json:"serverId"`
	Type      uint8     `json:"type"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
}

type createRes struct {
	ServerId         string `json:"serverId"`
	ServerName       string `json:"serverName"`
	Img              string `json:"img"`
	GeneralChannelId string `json:"generalChannelId"`
}

func UserServer(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userId := vars["userId"]
	var data []serverData

	tx, err := config.UseDBPool().DB.Begin()
	if err != nil {
		log.Println(err)
		return
	}

	defer func() {
		if err := config.UseDBPool().RollbackOrCommit(tx, err == nil); err != nil {
			log.Println("Error rolling back transaction:", err)
		}
	}()

	rows, err := tx.Query("SELECT * FROM server_members WHERE user_id = ?", userId)
	if err != nil {
		log.Println(err)
		http.Error(w, "Failed to execute query", http.StatusInternalServerError)
		return
	}

	defer func(rows *sql.Rows) {
		err := rows.Close()
		if err != nil {
			log.Println(err)
			return
		}
	}(rows)

	for rows.Next() {
		var membershipId int
		var serverId string
		var serverName string
		var img string
		var serverOwner bool
		var joinedAt time.Time
		var createdAt time.Time
		err = rows.Scan(&membershipId, &serverId, new(int), &serverOwner, &joinedAt)
		if err != nil {
			log.Println(err)
			http.Error(w, "Failed to scan row", http.StatusInternalServerError)
			return
		}

		err := tx.QueryRow("SELECT server_name, img, created_at FROM servers WHERE server_id = ? LIMIT 1", serverId).Scan(&serverName, &img, &createdAt)
		if err != nil {
			log.Println(err)
			http.Error(w, "Failed to execute query", http.StatusInternalServerError)
			return
		}

		if err != nil {
			log.Println(err)
			http.Error(w, "Failed to scan row", http.StatusInternalServerError)
			return
		}

		server := serverData{
			ServerId:  serverId,
			Name:      serverName,
			Img:       img,
			CreatedAt: createdAt,
			UserData: struct {
				JoinedAt     time.Time `json:"createdAt"`
				ServerOwner  bool      `json:"ServerOwner"`
				MembershipId int       `json:"membershipId"`
			}{
				JoinedAt:     joinedAt,
				ServerOwner:  serverOwner,
				MembershipId: membershipId,
			},
		}
		data = append(data, server)
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
	id := vars["serverId"]
	var data []channelData

	tx, err := config.UseDBPool().DB.Begin()
	if err != nil {
		return
	}

	defer func() {
		if err := config.UseDBPool().RollbackOrCommit(tx, err == nil); err != nil {
			log.Println("Error rolling back transaction:", err)
		}
	}()

	rows, err := tx.Query("SELECT * FROM channels WHERE server_id = ?", id)
	if err != nil {
		http.Error(w, "Failed to execute query", 500)
		return
	}

	defer func() {
		err := rows.Close()
		if err != nil {
			log.Println(err)
			http.Error(w, "Failed to close rows on the database", http.StatusInternalServerError)
			return
		}
	}()

	for rows.Next() {
		var id string
		var serverId string
		var channelType uint8
		var name string
		var createdAt time.Time
		err = rows.Scan(&id, &serverId, &channelType, &name, &createdAt)
		if err != nil {
			http.Error(w, "Failed to scan row", 500)
			return
		}

		row := channelData{
			Id:        id,
			ServerId:  serverId,
			Type:      channelType,
			Name:      name,
			CreatedAt: createdAt,
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
	var serverId = helper.GenerateUniqueId()
	log.Println("FD:", formData)
	var serverName = formData.Value["name"][0]
	var userId, _ = strconv.ParseInt(formData.Value["userId"][0], 10, 64)
	file, _, err := r.FormFile("img")
	var filename = strconv.Itoa(int(serverId)) + ".jpg"
	var generalChannelId = helper.GenerateUniqueId()
	var generalVoiceId = helper.GenerateUniqueId()
	var imgPath = filepath.Join(wd, "../", "public", "img", filename)
	var imgCdnPath = fmt.Sprintf("https://%s%s", r.Host, path.Join("/public/img/serverimage/", filename))

	if err != nil {
		log.Println("No file provided")
	} else {
		defer func(file multipart.File) {
			err := file.Close()
			if err != nil {
				log.Println(err)
				return
			}
		}(file)
		buf := bytes.NewBuffer(nil)
		if _, err := io.Copy(buf, file); err != nil {
			log.Println("Error copying file:", err)
			http.Error(w, "Internal Server Error", 500)
			return
		}
		err := os.WriteFile(imgPath, buf.Bytes(), 0644)
		if err != nil {
			log.Println(err)
			return
		}
	}

	tx, err := config.UseDBPool().DB.Begin()
	if err != nil {
		log.Println(err)
		return
	}

	defer func() {
		if err := config.UseDBPool().RollbackOrCommit(tx, err == nil); err != nil {
			log.Println("Error rolling back transaction:", err)
		}
	}()

	_, err = tx.Exec("INSERT INTO servers (server_name, server_id, img) VALUES (?, ?, ?)", serverName, serverId, imgCdnPath)
	if err != nil {
		log.Println(err)
		http.Error(w, "Failed add server into db", http.StatusInternalServerError)
		return
	} else {
		log.Printf("A new server has been added with ID %v, and the owner has been added. \n", serverId)
	}

	_, err = tx.Exec("INSERT INTO channels (server_id, channel_name, type, channel_id) VALUES (?, 'General', 1, ?), (?, 'General', 2, ?)", serverId, generalChannelId, serverId, generalVoiceId)
	if err != nil {
		log.Println(err)
		http.Error(w, "Failed add channels into database", http.StatusInternalServerError)
		return
	} else {
		log.Println(`Standard channels 'general' and 'general' (voice) have been created.`)
	}

	_, err = tx.Exec("INSERT INTO server_members ( server_id, user_id, server_owner) VALUES (?, ?, true)", serverId, userId)
	if err != nil {
		log.Println(err)
		http.Error(w, "Failed add user into database", http.StatusInternalServerError)
		return
	}

	var data = [1]createRes{{ServerId: strconv.Itoa(int(serverId)), ServerName: serverName, Img: imgCdnPath, GeneralChannelId: strconv.Itoa(int(generalChannelId))}}
	res, err := json.Marshal(data)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(res)
}

func CreateInviteLink(w http.ResponseWriter, r *http.Request) {
	var body map[string]interface{}
	log.Println(r)
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		log.Println(err)
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}
	serverId := body["serverId"].(string)
	randomString, err := helper.GenerateRandomString(8)
	inviteURL := "https://" + "localhost:3000" + "/invite/" + randomString
	if err != nil {
		log.Println(err)
		http.Error(w, "Error generating random String", http.StatusInternalServerError)
		return
	}

	tx, err := config.UseDBPool().DB.Begin()
	if err != nil {
		log.Println(err)
		return
	}

	defer func() {
		if err := config.UseDBPool().RollbackOrCommit(tx, err == nil); err != nil {
			log.Println("Error rolling back transaction:", err)
		}
	}()

	_, err = tx.Exec("INSERT INTO invite_links (invite_code, server_id) VALUES (?, ?)", randomString, serverId)
	if err != nil {
		log.Println(err)
		http.Error(w, "Error performing database operation", http.StatusInternalServerError)
		return
	}

	res, err := json.Marshal(map[string]interface{}{"inviteLink": inviteURL})
	if err != nil {
		http.Error(w, "Failed to encode JSON", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(res)
}

func JoinServer(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	inviteCode := vars["code"]
	userId, _ := strconv.ParseInt(vars["userId"], 10, 64)
	var serverName string
	var img string
	var createdAt time.Time
	var serverId string

	tx, err := config.UseDBPool().DB.Begin()
	if err != nil {
		log.Println(err)
		return
	}

	defer func() {
		if err := config.UseDBPool().RollbackOrCommit(tx, err == nil); err != nil {
			log.Println("Error rolling back transaction:", err)
		}
	}()

	err = tx.QueryRow("SELECT server_id, created_at FROM invite_links WHERE invite_code = ?", inviteCode).Scan(&serverId, &createdAt)
	if err != nil {
		log.Println(err)
		return
	}

	if time.Since(createdAt).Hours() > 7*24 {
		_, err = tx.Exec("DELETE FROM invite_links WHERE invite_code = ?", inviteCode)
		if err != nil {
			return
		}
		http.Error(w, "The invite code you used is no longer valid", http.StatusGone)
		return
	}

	log.Println("ID:", serverId)
	err = tx.QueryRow("SELECT server_id, server_name, img FROM servers WHERE server_id = ?", serverId).Scan(&serverId, &serverName, &img)
	if err != nil {
		log.Println("1", err)
		http.Error(w, "Failed to execute query", http.StatusInternalServerError)
		return
	}

	err = tx.QueryRow("SELECT user_id FROM server_members WHERE user_id = ? AND server_id = ?", userId, serverId).Scan(&userId)
	if err == nil {
		fmt.Println("User with ID", userId, "already exists on Server", serverId)
		return
	} else if !errors.Is(err, sql.ErrNoRows) {
		fmt.Println("Error checking user existence:", err)
		http.Error(w, "The user is already on the server", http.StatusConflict)
		return
	}

	log.Println("INSERTING INTO server_members")
	_, err = tx.Exec("INSERT INTO server_members ( server_id, user_id, server_owner) VALUES (?,?,?)", serverId, userId, false)

	var joinedAt time.Time
	var serverOwner bool
	var membershipId int

	err = tx.QueryRow("SELECT joined_at, server_owner, membership_id FROM server_members WHERE server_id = ? AND user_id = ?", serverId, userId).Scan(&joinedAt, &serverOwner, &membershipId)
	if err != nil {
		log.Println("2", err)
		http.Error(w, "Failed to execute query", http.StatusInternalServerError)
		return
	}

	server := serverData{
		ServerId:  serverId,
		Name:      serverName,
		Img:       img,
		CreatedAt: createdAt,
		UserData: struct {
			JoinedAt     time.Time `json:"createdAt"`
			ServerOwner  bool      `json:"ServerOwner"`
			MembershipId int       `json:"membershipId"`
		}{
			JoinedAt:     joinedAt,
			ServerOwner:  serverOwner,
			MembershipId: membershipId,
		},
	}

	res, err := json.Marshal(server)
	if err != nil {
		http.Error(w, "Failed to encode JSON", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(res)
}

func ServerMembers(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	serverId := vars["serverId"]
	var data []userData

	tx, err := config.UseDBPool().DB.Begin()
	if err != nil {
		log.Println(err)
		return
	}

	defer func() {
		if err := config.UseDBPool().RollbackOrCommit(tx, err == nil); err != nil {
			log.Println("Error rolling back transaction:", err)
		}
	}()

	rows, err := tx.Query("SELECT users.user_id, users.username, users.display_name, users.appearance, users.bio, users.status, users.last_seen, users.joined_at, users.pronouns, users.img_url, users.online FROM server_members JOIN users ON server_members.user_id = users.user_id WHERE server_members.server_id = ?", serverId)
	if err != nil {
		return
	}

	for rows.Next() {
		var userId string
		var username string
		var displayName string
		var appearance uint8
		var bio string
		var status string
		var joinedAt time.Time
		var lastSeen time.Time
		var pronouns string
		var img string
		var online bool

		err := rows.Scan(&userId, &username, &displayName, &appearance, &bio, &status, &lastSeen, &joinedAt, &pronouns, &img, &online)
		if err != nil {
			log.Println(err)
			http.Error(w, "Failed to scan row", http.StatusInternalServerError)
			return
		}

		user := userData{UserId: userId, Username: username, DisplayName: displayName, Appearance: appearance, Bio: bio, JoinedAt: joinedAt, LastSeen: lastSeen, Pronouns: pronouns, Img: img, Online: online}

		data = append(data, user)
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
