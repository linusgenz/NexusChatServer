package api

import (
	"database/sql"
	"encoding/json"
	"github.com/dgrijalva/jwt-go"
	"golang.org/x/crypto/bcrypt"
	"log"
	"net/http"
	"strconv"
	"time"
	"webserver/internal/config"
	"webserver/internal/helper"
)

type user struct {
	Id             int64  `json:"id"`
	Username       string `json:"username"`
	Email          string `json:"email"`
	Password       string `json:"password"`
	hashedPassword []byte
	ImgUrl         string `json:"imgUrl"`
	DisplayName    string `json:"displayName"`
}

type JWTClaims struct {
	UserID   int64  `json:"userId"`
	Username string `json:"username"`
	jwt.StandardClaims
}

type authResponse struct {
	Token       string `json:"token"`
	DisplayName string `json:"displayName"`
	Username    string `json:"username"`
	UserId      string `json:"userId"`
	Img         string `json:"img"`
}

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	var userCredentials user

	if err := json.NewDecoder(r.Body).Decode(&userCredentials); err != nil {
		log.Println(err, r.Body, userCredentials)
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	log.Println("userCredentials", userCredentials)

	ok, user, err := validateUserCredentials(userCredentials)
	if err != nil {
		log.Print(err)
		http.Error(w, "Error validating websocket credentials", http.StatusInternalServerError)
		return
	}
	if !ok {
		http.Error(w, "Invalid username or password", http.StatusUnauthorized)
		return
	}

	token, err := generateJWTToken(userCredentials)
	if err != nil {
		http.Error(w, "Error generating JWT token", http.StatusInternalServerError)
		return
	}

	var data = authResponse{Token: token, DisplayName: user.DisplayName, Username: user.Username, UserId: strconv.FormatInt(user.Id, 10), Img: user.ImgUrl}
	res, err := json.Marshal(data)
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(res)
	if err != nil {
		return
	}
}

func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	var userData user
	if err := json.NewDecoder(r.Body).Decode(&userData); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}
	log.Println(r.Body)
	log.Println(userData)

	ok, err, user := addUserToDb(userData)
	if err != nil {
		log.Print(err)
		http.Error(w, "Error adding user to db", http.StatusInternalServerError)
		return
	}
	if !ok {
		http.Error(w, "User already present", http.StatusConflict)
		return
	}

	token, err := generateJWTToken(user)
	if err != nil {
		http.Error(w, "Error generating JWT token", http.StatusInternalServerError)
		return
	}

	var data = authResponse{Token: token, DisplayName: user.DisplayName, Username: user.Username, UserId: strconv.FormatInt(user.Id, 10), Img: user.ImgUrl}
	res, err := json.Marshal(data)
	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(res)
	if err != nil {
		return
	}
}

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenString := r.Header.Get("Authorization")
		if tokenString == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
			return config.JwtKey, nil
		})

		if err != nil || !token.Valid {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func validateUserCredentials(credentials user) (bool, user, error) {
	tx, err := config.UseDBPool().DB.Begin()
	if err != nil {
		log.Println(err)
		return false, user{}, err
	}

	defer func() {
		if err := config.UseDBPool().RollbackOrCommit(tx, err == nil); err != nil {
			log.Println("Error rolling back transaction:", err)
		}
	}()

	rows, err := tx.Query("SELECT user_id, username, password, display_name FROM users WHERE username = ? LIMIT 1", credentials.Username)

	defer func(rows *sql.Rows) {
		err := rows.Close()
		if err != nil {
			log.Println(err)
		}
	}(rows)

	if err != nil {
		return false, user{}, err
	}
	var userId int64
	var username string
	var password []byte
	var displayName string

	for rows.Next() {
		err = rows.Scan(&userId, &username, &displayName, &password)
	}

	log.Println(userId, username, displayName, password)

	var passwordIsValid = bcrypt.CompareHashAndPassword(password, []byte(credentials.Password))

	if passwordIsValid != nil {
		return false, user{}, nil
	}
	return true, user{Id: userId, Username: username, DisplayName: displayName}, nil
}

func addUserToDb(userData user) (bool, error, user) {
	userData.Id = helper.GenerateUniqueId()
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(userData.Password), bcrypt.DefaultCost)
	userData.hashedPassword = hashedPassword
	if err != nil {
		return false, err, user{}
	}
	//if value == nil {
	userData.ImgUrl = "https://" + config.HOST + config.PORT + "/public/img/user_default.jpg"
	//} else {
	//	websocket.Img_url = "https://localhost:3300/public/img/user/" + strconv.Itoa(int(websocket.Id)) + ".jpg"
	//}

	tx, err := config.UseDBPool().DB.Begin()
	if err != nil {
		log.Println(err)
		return false, err, user{}
	}

	defer func() {
		if err := config.UseDBPool().RollbackOrCommit(tx, err == nil); err != nil {
			log.Println("Error rolling back transaction:", err)
		}
	}()

	exists, err := checkIfUserExists(tx, userData.Username, userData.Email)
	if err != nil {
		log.Print(err)
	}
	if exists {
		return false, err, user{}
	}

	_, err = tx.Exec("INSERT INTO users (user_id, username, display_name, email, language, status, appearance, img_url, password) VALUES (?,?,?,?,'en',null,1,?, ?)", userData.Id, userData.Username, userData.DisplayName, userData.Email, userData.ImgUrl, userData.hashedPassword)
	if err != nil {
		return false, err, user{}
	}
	return true, nil, user{Id: userData.Id, Username: userData.Username, DisplayName: userData.DisplayName}
}

func checkIfUserExists(tx *sql.Tx, username, email string) (bool, error) {
	query := "SELECT COUNT(*) FROM users WHERE username = ? OR email = ?"

	var count int
	err := tx.QueryRow(query, username, email).Scan(&count)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

func generateJWTToken(user user) (string, error) {
	claims := JWTClaims{
		UserID:   user.Id,
		Username: user.Username,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(time.Hour * 24).Unix(), // Token expires in 24 hours
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(config.JwtKey)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}
