package main

import (
	"database/sql"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"

	"github.com/matthewdale/manualsmap.com/mapblocks"
	"github.com/matthewdale/manualsmap.com/tokens"
)

const secret = ``
const teamID = ""
const keyID = ""
const origin = ""
const psqlConn = "postgres://postgres:@localhost:5432/manualsmap?sslmode=disable"

func main() {
	router := mux.NewRouter()

	tokensSvc, err := tokens.NewService(teamID, keyID, []byte(secret))
	if err != nil {
		log.Fatal("Error parsing private key PEM file", err)
	}
	router.Methods("GET").Path("/token").Handler(tokens.GetHandler(tokensSvc))

	db, err := sql.Open("postgres", psqlConn)
	if err != nil {
		log.Fatal("Error connecting to Postgres DB", err)
	}
	mapBlocksSvc := mapblocks.NewService(db)
	router.Methods("GET").Path("/mapblocks").Handler(mapblocks.GetHandler(mapBlocksSvc))
	router.Methods("GET").Path("/mapblocks/{id}/cars").Handler(mapblocks.GetCarsHandler(mapBlocksSvc))
	router.Methods("POST").Path("/cars").Handler(mapblocks.PostCarsHandler(mapBlocksSvc))

	// TODO: Remove or replace?
	router.PathPrefix("/").Handler(http.FileServer(http.Dir("public")))

	if err := http.ListenAndServe(":8080", router); err != nil {
		log.Print("Error starting HTTP server:", err)
	}
}
