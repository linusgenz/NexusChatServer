package api

import (
	"database/sql"
	"encoding/json"
	"github.com/dgrijalva/jwt-go"
	"golang.org/x/crypto/bcrypt"
	"log"
	"net/http"
	"os"
	//"strconv"
	"fmt"
	"time"
	"webserver/internal/helper"
)

var secretKey = []byte(os.Getenv("JWT_SECRET_KEY"))
var PORT = os.Getenv("PORT")
var HOST = os.Getenv("HOST")

type user struct {
	Id             int64  `json:"id"`
	Username       string `json:"username"`
	Email          string `json:"email"`
	Password       string `json:"password"`
	hashedPassword []byte
	Img_url        string `json:"img_url"`
	Display_name   string `json:"display_name"`
}

type JWTClaims struct {
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
	jwt.StandardClaims
}

type authResponse struct {
	Token        string `json:"token"`
	Display_name string `json:"display_name"`
	Username     string `json:"username"`
}

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	var userCredentials user

	if err := json.NewDecoder(r.Body).Decode(&userCredentials); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	ok, user, err := validateUserCredentials(userCredentials)
	if err != nil {
		log.Print(err)
		http.Error(w, "Error validating user credentials", http.StatusInternalServerError)
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

	var data = authResponse{Token: token, Display_name: user.Display_name, Username: user.Username}
	res, err := json.Marshal(data)
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	w.Write(res)
}

func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	var userData user
	if err := json.NewDecoder(r.Body).Decode(&userData); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	ok, err := addUserToDb(userData)
	if err != nil {
		log.Print(err)
		http.Error(w, "Error adding user to db", http.StatusInternalServerError)
		return
	}
	if !ok {
		http.Error(w, "User already present", http.StatusConflict)
		return
	}

	token, err := generateJWTToken(userData)
	if err != nil {
		http.Error(w, "Error generating JWT token", http.StatusInternalServerError)
		return
	}

	var data = authResponse{Token: token, Display_name: userData.Display_name, Username: userData.Username}
	res, err := json.Marshal(data)
	w.Header().Set("Content-Type", "application/json")
	w.Write(res)
}

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenString := r.Header.Get("Authorization")
		if tokenString == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
			return secretKey, nil
		})

		if err != nil || !token.Valid {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func validateUserCredentials(creds user) (bool, user, error) {
	db, err := sql.Open("sqlite3", "../data/data.sqlite")
	if err != nil {
		return false, user{}, err
	}

	rows, err := db.Query("SELECT * FROM users WHERE username = ?", creds.Username)

	if err != nil {
		return false, user{}, err
	}
	var user_id int64
	var username string
	var password []byte
	var url string
	var display_name string
	var email string
	var created_at time.Time

	for rows.Next() {
		err = rows.Scan(&user_id, &username, &display_name, &email, &url, &password, &created_at)
	}

	var passwordIsValid = bcrypt.CompareHashAndPassword(password, []byte(creds.Password))

	if passwordIsValid != nil {
		return false, user{}, nil
	}
	defer rows.Close()
	return true, user{Id: user_id, Username: username, Display_name: display_name}, nil
}

func addUserToDb(user user) (bool, error) {
	user.Id = helper.GenerateUniqueId()
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	user.hashedPassword = hashedPassword
	if err != nil {
		return false, err
	}
	//if value == nil {
	user.Img_url = fmt.Sprintf("https://%s%s/public/img/user/default.jpg", HOST, PORT)
	//} else {
	//	user.Img_url = "https://localhost:3300/public/img/user/" + strconv.Itoa(int(user.Id)) + ".jpg"
	//}

	db, err := sql.Open("sqlite3", "../data/data.sqlite")
	if err != nil {
		return false, err
	}

	exists, err := checkIfUserExists(db, user.Username, user.Email)
	if err != nil {
		log.Print(err)
	}
	if exists {
		return false, err
	}

	db.Exec("INSERT INTO users (user_id, username, display_name, email, language, status, bio, custom_status, img_url, password) VALUES (?,?,?,?,'en',1,NULL, NULL,?, ?)", user.Id, user.Username, user.Display_name, user.Email, user.Img_url, user.hashedPassword)
	return true, nil
}

func checkIfUserExists(db *sql.DB, username, email string) (bool, error) {
	query := "SELECT COUNT(*) FROM users WHERE username = ? OR email = ?"

	var count int
	err := db.QueryRow(query, username, email).Scan(&count)
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
	tokenString, err := token.SignedString(secretKey)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func signout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    "",
		Expires:  time.Unix(0, 0),
		Path:     "/",
		HttpOnly: true,
	})

	responseMessage := map[string]string{"message": "You've been signed out!"}

	responseJSON, err := json.Marshal(responseMessage)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	w.Write(responseJSON)
}
