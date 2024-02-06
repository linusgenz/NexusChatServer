package config

import (
	"github.com/joho/godotenv"
	"log"
	"os"
)

var HOST string
var PORT string
var JwtKey []byte

func LoadConfig() {
	if err := godotenv.Load("../.env"); err != nil {
		log.Fatal("Error loading .env file", err)
	}
	HOST = os.Getenv("HOST")
	PORT = os.Getenv("PORT")
	JwtKey = []byte(os.Getenv("JWT_SECRET_KEY"))

}
