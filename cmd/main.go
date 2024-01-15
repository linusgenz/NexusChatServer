package main

import (
	"crypto/tls"
	"log"
	"net/http"
	"webserver/internal/api"
	"webserver/internal/webrtc"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/jiyeyuran/mediasoup-go"
)

func main() {
	port := ":3300"
	router := mux.NewRouter()

	worker, err := mediasoup.NewWorker(
		mediasoup.WithLogLevel("debug"),
		mediasoup.WithRtcMinPort(40000),
		mediasoup.WithRtcMaxPort(49999),
	)

	if err != nil {
		log.Fatal("Mediasoup worker error:", err)
	}
	worker.On("died", func(err error) {
		log.Fatalf("mediasoup worker has died %v\n", err)
	})

	headersOk := handlers.AllowedHeaders([]string{"Content-Type", "Authorization"})
	originsOk := handlers.AllowedOrigins([]string{"*"})
	methodsOk := handlers.AllowedMethods([]string{"GET", "HEAD", "POST", "PUT", "OPTIONS"})
	handler := handlers.CORS(headersOk, originsOk, methodsOk)(router)

	apiRouter := router.PathPrefix("/api").Subrouter()
	apiRouter.HandleFunc("/list", api.List).Methods("GET")
	apiRouter.HandleFunc("/{serverid}/channels", api.Channels).Methods("GET")
	apiRouter.HandleFunc("/create", api.Create).Methods("POST")

	router.PathPrefix("/public/").Handler(http.StripPrefix("/public/", http.FileServer(http.Dir("public"))))

	log.Printf("Go server running at port %v \n", port)
	router.Handle("/", handler)
	router.HandleFunc("/mediasoup", webrtc.HandleWebSocketConnections)

	tlsConfig := &tls.Config{
		MinVersion:               tls.VersionTLS12,
		CurvePreferences:         []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
		PreferServerCipherSuites: true,
		InsecureSkipVerify:       true, // Set to false in production
	}

	server := &http.Server{
		Addr:      port,
		Handler:   handler,
		TLSConfig: tlsConfig,
	}

	server.ListenAndServeTLS("./ssl/cert.pem", "./ssl/key.pem")
}
