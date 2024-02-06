package main

import (
	"crypto/tls"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"webserver/internal/api"
	"webserver/internal/config"
	"webserver/internal/webrtc"
	"webserver/internal/websocket"
)

func main() {
	config.LoadConfig()
	//croc.DeleteExpiredInviteLinks()

	dbConfig := config.DatabaseConfig{
		Driver:   "sqlite3",
		Source:   "../data/data.sqlite",
		MaxConns: 50,
	}

	pool, err := config.NewDatabasePool(dbConfig)
	if err != nil {
		log.Fatal(err)
	}

	defer pool.DB.Close()

	config.InitDatabase(pool)

	router := mux.NewRouter()

	headersOk := handlers.AllowedHeaders([]string{"Content-Type", "Authorization"})
	originsOk := handlers.AllowedOrigins([]string{"*"})
	methodsOk := handlers.AllowedMethods([]string{"GET", "HEAD", "POST", "PUT", "OPTIONS"})
	handler := handlers.CORS(headersOk, originsOk, methodsOk)(router)

	apiRouter := router.PathPrefix("/api").Subrouter()
	createRouter := apiRouter.PathPrefix("/create").Subrouter()
	createRouter.HandleFunc("/server", api.Create).Methods("POST")
	createRouter.HandleFunc("/invitelink", api.CreateInviteLink).Methods("POST")
	apiRouter.HandleFunc("/{userId}/server", api.UserServer).Methods("GET")
	apiRouter.HandleFunc("/{serverId}/channels", api.Channels).Methods("GET")
	apiRouter.HandleFunc("/{serverId}/members", api.ServerMembers).Methods("GET")
	apiRouter.HandleFunc("/{userId}/joinServer/{inviteId}", api.JoinServer).Methods("GET")
	apiRouter.HandleFunc("/auth/login", api.LoginHandler).Methods("POST")
	apiRouter.HandleFunc("/auth/register", api.RegisterHandler).Methods("POST")

	router.HandleFunc("/invite/{code}/{userId}", api.JoinServer).Methods("GET")

	router.PathPrefix("/public/").Handler(http.StripPrefix("/public/", http.FileServer(http.Dir("../public"))))

	router.Handle("/", handler)

	router.HandleFunc("/webrtc", webrtc.HandleWebSocketConnections)
	router.HandleFunc("/wss", websocket.HandleWebSocketConnections)

	tlsConfig := &tls.Config{
		MinVersion:               tls.VersionTLS12,
		CurvePreferences:         []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
		PreferServerCipherSuites: true,
		InsecureSkipVerify:       true, // Set to false in production
	}

	server := &http.Server{
		Addr:      config.PORT,
		Handler:   handler,
		TLSConfig: tlsConfig,
	}

	log.Printf("Go server running at port %v \n", config.PORT)
	err = server.ListenAndServeTLS("../ssl/cert.pem", "../ssl/key.pem")
	if err != nil {
		log.Fatal("Could not start https server", err)
	}
}
