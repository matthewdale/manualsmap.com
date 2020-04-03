package main

import (
	"database/sql"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/matthewdale/manualsmap.com/services"
	"github.com/shopspring/decimal"

	"github.com/alecthomas/kong"
	"github.com/dpapathanasiou/go-recaptcha"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"

	"github.com/matthewdale/manualsmap.com/handlers/images"
	"github.com/matthewdale/manualsmap.com/handlers/mapblocks"
	"github.com/matthewdale/manualsmap.com/handlers/mapkit"
)

var opts struct {
	// Server configuration.
	Addr string `kong:"name='addr',default=':8080',help='the address to listen on'"`

	// Mapkit JS configuration.
	AppleTeamID  string `kong:"required,name='apple-team-id',help='Apple developer team ID'"`
	MapkitKeyID  string `kong:"required,name='mapkit-key-id',help='Apple Mapkit key ID'"`
	MapkitSecret string `kong:"required,name='mapkit-secret',help='path to the Apple Mapkit P8/PEM secret file'"`
	MapkitOrigin string `kong:"name='mapkit-origin',help='Apple Mapkit JWT origin domain'"`

	// reCAPTCHA API configuration.
	RecaptchaSecret string `kong:"required,name='recaptcha-secret',help='reCAPTCHA API secret'"`

	// Cloudinary API configuration.
	CloudinarySecret string `kong:"required,name='cloudinary-secret',help='Cloudinary API secret'"`

	// Postgres connection.
	PSQLConn string `kong:"required,name='psql-conn',help='Postgres SQL connection string'"`

	LicenseSalt string `kong:"required,name='license-salt',help='salt for hashed license plate information'"`
}

func main() {
	// Marshal decimal types as numbers, not strings.
	decimal.MarshalJSONWithoutQuotes = true
	kong.Parse(&opts, kong.UsageOnError())
	recaptcha.Init(opts.RecaptchaSecret)

	mapkitSecret, err := ioutil.ReadFile(opts.MapkitSecret)
	if err != nil {
		log.Fatal("Error reading secret key", err)
	}

	appleMapkit, err := services.NewAppleMapkit(
		opts.AppleTeamID,
		opts.MapkitKeyID,
		mapkitSecret,
		opts.MapkitOrigin)
	if err != nil {
		log.Fatal("Error parsing private key PEM file", err)
	}

	db, err := sql.Open("postgres", opts.PSQLConn)
	if err != nil {
		log.Fatal("Error connecting to Postgres DB", err)
	}

	persistence := services.NewPersistence(db, []byte(opts.LicenseSalt))
	cloudinary := services.NewCloudinary(opts.CloudinarySecret)

	router := mux.NewRouter()
	router.
		Methods("GET").
		Path("/mapkit/token").
		Handler(mapkit.GetTokenHandler(appleMapkit))
	router.
		Methods("POST").
		Path("/images/signature").
		Handler(images.PostSignatureHandler(cloudinary))
	router.
		Methods("POST").
		Path("/images/notification").
		Handler(images.PostNotificationHandler(persistence, cloudinary))
	router.
		Methods("GET").
		Path("/mapblocks").
		Handler(mapblocks.GetHandler(persistence))
	router.
		Methods("GET").
		Path("/mapblocks/{id}/cars").
		Handler(mapblocks.GetCarsHandler(persistence, cloudinary))
	router.
		Methods("POST").
		Path("/cars").
		Handler(mapblocks.PostCarsHandler(persistence))
	router.
		PathPrefix("/").
		Handler(http.FileServer(http.Dir("public")))

	if err := http.ListenAndServe(opts.Addr, router); err != nil {
		log.Print("Error starting HTTP server:", err)
	}
}
