package main

import (
	"crypto/tls"
	"log"
	"net/http"
	"os"
	"webserver/internal/api"
	"webserver/internal/webrtc"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load("../.env"); err != nil {
		log.Fatal("Error loading .env file", err)
	}

	PORT := os.Getenv("PORT")
	router := mux.NewRouter()

	headersOk := handlers.AllowedHeaders([]string{"Content-Type", "Authorization"})
	originsOk := handlers.AllowedOrigins([]string{"*"})
	methodsOk := handlers.AllowedMethods([]string{"GET", "HEAD", "POST", "PUT", "OPTIONS"})
	handler := handlers.CORS(headersOk, originsOk, methodsOk)(router)

	apiRouter := router.PathPrefix("/api").Subrouter()
	apiRouter.HandleFunc("/list", api.List).Methods("GET")
	apiRouter.HandleFunc("/{serverid}/channels", api.Channels).Methods("GET")
	apiRouter.HandleFunc("/create", api.Create).Methods("POST")
	apiRouter.HandleFunc("/auth/login", api.LoginHandler).Methods("POST")
	apiRouter.HandleFunc("/auth/register", api.RegisterHandler).Methods("POST")
	router.PathPrefix("/public/").Handler(http.StripPrefix("/public/", http.FileServer(http.Dir("../public"))))

	log.Printf("Go server running at port %v \n", PORT)
	router.Handle("/", handler)
	router.HandleFunc("/wss", webrtc.HandleWebSocketConnections)

	tlsConfig := &tls.Config{
		MinVersion:               tls.VersionTLS12,
		CurvePreferences:         []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
		PreferServerCipherSuites: true,
		InsecureSkipVerify:       true, // Set to false in production
	}

	server := &http.Server{
		Addr:      PORT,
		Handler:   handler,
		TLSConfig: tlsConfig,
	}

	server.ListenAndServeTLS("../ssl/cert.pem", "../ssl/key.pem")
}
