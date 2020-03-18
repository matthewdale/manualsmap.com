package main

import (
	"context"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"log"
	"net/http"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/go-kit/kit/endpoint"
	httptransport "github.com/go-kit/kit/transport/http"
	"github.com/pkg/errors"
)

const secret = ``
const teamID = ""
const keyID = ""
const origin = ""

func main() {
	http.Handle("/", http.FileServer(http.Dir("public")))

	svc := apiService{}
	tokenHandler := httptransport.NewServer(
		makeTokenEndpoint(svc),
		func(_ context.Context, r *http.Request) (interface{}, error) { return nil, nil },
		encodeResponse,
	)
	http.Handle("/token", tokenHandler)

	carsHandler := httptransport.NewServer(
		makeCarsEndpoint(svc),
		func(_ context.Context, r *http.Request) (interface{}, error) { return nil, nil },
		encodeResponse,
	)
	http.Handle("/cars", carsHandler)

	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Print("Error starting HTTP server:", err)
	}
}

type apiService struct{}

type HTTPCodedResponse interface {
	StatusCode() int
}

func encodeResponse(_ context.Context, writer http.ResponseWriter, response interface{}) error {
	if res, ok := response.(HTTPCodedResponse); ok {
		if code := res.StatusCode(); code > 0 {
			writer.WriteHeader(res.StatusCode())
		}
	}
	writer.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(writer).Encode(response)
}

func (apiService) Token() (string, error) {
	block, _ := pem.Decode([]byte(secret))
	if block == nil || block.Type != "PRIVATE KEY" {
		return "", errors.New("invalid private key block")
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return "", errors.WithMessage(err, "error parsing private key")
	}

	token := jwt.NewWithClaims(jwt.SigningMethodES256, jwt.MapClaims{
		"iss": teamID,
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(1 * time.Hour).Unix(),
		// "origin": origin,
	})
	token.Header["kid"] = keyID

	tokenString, err := token.SignedString(key)
	if err != nil {
		return "", errors.WithMessage(err, "error generating signed JWT")
	}
	return tokenString, nil
}

type tokenResponse struct {
	Token      string `json:"token,omitempty"`
	Err        string `json:"err,omitempty"`
	statusCode int
}

func (res tokenResponse) StatusCode() int {
	return res.statusCode
}

func makeTokenEndpoint(svc apiService) endpoint.Endpoint {
	return func(_ context.Context, request interface{}) (interface{}, error) {
		token, err := svc.Token()
		if err != nil {
			return tokenResponse{Err: err.Error(), statusCode: http.StatusInternalServerError}, nil
		}
		return tokenResponse{Token: token}, nil
	}
}

type car struct {
	ID        string  `json:"id,omitempty"`
	Year      int     `json:"year,omitempty"`
	Brand     string  `json:"brand,omitempty"`
	Model     string  `json:"model,omitempty"`
	Trim      string  `json:"trim,omitempty"`
	Color     string  `json:"color,omitempty"`
	Longitude float64 `json:"longitude,omitempty"`
	Latitude  float64 `json:"latitude,omitempty"`
}

type carsResponse struct {
	Cars       []car  `json:"cars,omitempty"`
	Err        string `json:"err,omitempty"`
	statusCode int
}

func (res carsResponse) StatusCode() int {
	return res.statusCode
}

func (apiService) Cars() ([]car, error) {
	return []car{
		{
			ID:        "NP12108",
			Year:      2006,
			Brand:     "Audi",
			Model:     "A4",
			Trim:      "2.0T",
			Color:     "black",
			Latitude:  47.6249008,
			Longitude: -122.3469808,
		},
		{
			ID:        "BPW3515",
			Year:      1989,
			Brand:     "Toyota",
			Model:     "Supra",
			Color:     "white",
			Latitude:  47.6248985,
			Longitude: -122.346979,
		},
		{
			ID:        "417-YWR",
			Year:      2006,
			Brand:     "Subaru",
			Model:     "Outback",
			Color:     "gunmetal",
			Latitude:  47.6286267,
			Longitude: -122.3439477,
		},
		{
			ID:        "ARN6500",
			Year:      2005,
			Brand:     "Honda",
			Model:     "Civic",
			Trim:      "LX",
			Color:     "white",
			Latitude:  47.6285353,
			Longitude: -122.3441541,
		},
		{
			ID:        "BKU2709",
			Year:      1991,
			Brand:     "Suzuki",
			Model:     "Cappucino",
			Color:     "black",
			Latitude:  47.6279952,
			Longitude: -122.3453595,
		},
		{
			ID:        "BSE4329",
			Year:      2019,
			Brand:     "Ford",
			Model:     "Fiesta",
			Trim:      "ST",
			Color:     "orange",
			Latitude:  47.6286892,
			Longitude: -122.3438791,
		},
		{
			ID:        "BMX7109",
			Year:      2012,
			Brand:     "Honda",
			Model:     "Civic",
			Trim:      "Si",
			Color:     "black",
			Latitude:  47.6035104,
			Longitude: -122.1317393,
		},
	}, nil
}

func makeCarsEndpoint(svc apiService) endpoint.Endpoint {
	return func(_ context.Context, request interface{}) (interface{}, error) {
		cars, err := svc.Cars()
		if err != nil {
			return carsResponse{Err: err.Error(), statusCode: http.StatusInternalServerError}, nil
		}
		return carsResponse{Cars: cars}, nil
	}
}
