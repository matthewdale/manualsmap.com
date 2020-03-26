package main

import (
	"database/sql"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/alecthomas/kong"
	"github.com/dpapathanasiou/go-recaptcha"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"

	"github.com/matthewdale/manualsmap.com/mapblocks"
	"github.com/matthewdale/manualsmap.com/tokens"
)

var opts struct {
	// Server configuration.
	Addr string `kong:"name='addr',default=':8080',help='the address to listen on'"`

	// Mapkit JS configuration.
	AppleTeamID  string `kong:"required,name='apple-team-id',help='Apple developer team ID'"`
	MapkitKeyID  string `kong:"required,name='mapkit-key-id',help='Apple Mapkit key ID'"`
	MapkitKey    string `kong:"required,name='mapkit-key',help='path to the Apple Mapkit P8/PEM secret file'"`
	MapkitOrigin string `kong:"name='mapkit-origin',help='Apple Mapkit JWT origin domain'"`

	// reCAPTCHA API configuration.
	RecaptchaKey string `kong:"required,name='recaptcha-key',help='reCAPTCHA secret key'"`

	// Postgres connection.
	PSQLConn string `kong:"required,name='psql-conn',help='Postgres SQL connection string'"`

	LicenseSalt string `kong:"required,name='license-salt',help='salt for hashed license plate information'"`
}

func main() {
	kong.Parse(&opts, kong.UsageOnError())

	recaptcha.Init(opts.RecaptchaKey)

	router := mux.NewRouter()

	mapkitSecret, err := ioutil.ReadFile(opts.MapkitKey)
	if err != nil {
		log.Fatal("Error reading secret key", err)
	}
	tokensSvc, err := tokens.NewService(opts.AppleTeamID, opts.MapkitKeyID, mapkitSecret, opts.MapkitOrigin)
	if err != nil {
		log.Fatal("Error parsing private key PEM file", err)
	}
	router.Methods("GET").Path("/token").Handler(tokens.GetHandler(tokensSvc))

	db, err := sql.Open("postgres", opts.PSQLConn)
	if err != nil {
		log.Fatal("Error connecting to Postgres DB", err)
	}
	mapBlocksSvc := mapblocks.NewService(db, []byte(opts.LicenseSalt))
	router.Methods("GET").Path("/mapblocks").Handler(mapblocks.GetHandler(mapBlocksSvc))
	router.Methods("GET").Path("/mapblocks/{id}/cars").Handler(mapblocks.GetCarsHandler(mapBlocksSvc))
	router.Methods("POST").Path("/cars").Handler(mapblocks.PostCarsHandler(mapBlocksSvc))

	// TODO: Remove or replace?
	router.PathPrefix("/").Handler(http.FileServer(http.Dir("public")))

	if err := http.ListenAndServe(opts.Addr, router); err != nil {
		log.Print("Error starting HTTP server:", err)
	}
}
