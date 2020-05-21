package mapblocks

import (
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"strconv"

	"github.com/go-kit/kit/endpoint"
	httptransport "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"github.com/xeipuuv/gojsonschema"

	"github.com/matthewdale/manualsmap.com/encoders"
	"github.com/matthewdale/manualsmap.com/middlewares"
	"github.com/matthewdale/manualsmap.com/services"
)

type getCarsRequest struct {
	mapBlockID int
}

type carResponse struct {
	Year         int    `json:"year"`
	Make         string `json:"make"`
	Model        string `json:"model"`
	Trim         string `json:"trim"`
	Color        string `json:"color"`
	ImageURL     string `json:"imageUrl"`
	ThumbnailURL string `json:"thumbnailUrl"`
}

type getCarsResponse struct {
	Cars []carResponse `json:"cars"`
}

func getCarsEndpoint(persistence services.Persistence, cloudinary services.Cloudinary) endpoint.Endpoint {
	return func(_ context.Context, request interface{}) (interface{}, error) {
		cars, err := persistence.GetCars(request.(getCarsRequest).mapBlockID)
		if err != nil {
			return nil, encoders.NewJSONError(
				errors.WithMessage(err, "error getting car"),
				http.StatusInternalServerError)
		}

		carResponses := make([]carResponse, 0, len(cars))
		for _, car := range cars {
			carResponses = append(carResponses, carResponse{
				Year:         car.Year,
				Make:         car.Make,
				Model:        car.Model,
				Trim:         car.Trim,
				Color:        car.Color,
				ImageURL:     cloudinary.URL(car.Image, "").String(),
				ThumbnailURL: cloudinary.URL(car.Image, "c_limit,w_300").String(),
			})
		}
		return getCarsResponse{Cars: carResponses}, nil
	}
}

func GetCarsHandler(persistence services.Persistence, cloudinary services.Cloudinary) http.Handler {
	return httptransport.NewServer(
		getCarsEndpoint(persistence, cloudinary),
		func(_ context.Context, r *http.Request) (interface{}, error) {
			vars := mux.Vars(r)
			id, ok := vars["id"]
			if !ok {
				return nil, encoders.NewJSONError(
					errors.New("invalid request, missing {id} in path"),
					http.StatusBadRequest)
			}
			mapBlockID, err := strconv.Atoi(id)
			if err != nil {
				return nil, encoders.NewJSONError(
					errors.WithMessage(err, "invalid {id} format, must be integer"),
					http.StatusBadRequest)
			}
			return getCarsRequest{mapBlockID: mapBlockID}, nil
		},
		encoders.JSONResponseEncoder,
	)
}

func GetCarsSchemaHandler() http.Handler {
	return httptransport.NewServer(
		func(_ context.Context, request interface{}) (interface{}, error) {
			return postCarsRequestSchema, nil
		},
		func(_ context.Context, r *http.Request) (interface{}, error) {
			return nil, nil
		},
		encoders.JSONResponseEncoder,
	)
}

var postCarsRequestValidator *gojsonschema.Schema
var postCarsRequestSchema = map[string]interface{}{
	"$schema": "http://json-schema.org/draft-07/schema#",
	"type":    "object",
	"properties": map[string]interface{}{
		"licenseState": map[string]interface{}{
			"type": "string",
			"enum": []string{
				"AL", "AK", "AZ", "ZR", "CA", "CO", "CT", "DE", "FL", "GA", "HI", "ID", "IL",
				"IN", "IA", "KS", "KY", "LA", "ME", "MD", "MA", "MI", "MN", "MS", "MO", "MT",
				"NE", "NV", "NH", "NJ", "NM", "NY", "NC", "ND", "OH", "OK", "OR", "PA", "RI",
				"SC", "SD", "TN", "TX", "UT", "VT", "VA", "WA", "DC", "WV", "WI", "WY",
			},
		},
		"licensePlate": map[string]interface{}{
			"type":      "string",
			"minLength": 3,
			"maxLength": 20,
		},
		"year": map[string]interface{}{
			"type":    "number",
			"minimum": 1900,
			"maximum": 2100,
		},
		"make": map[string]interface{}{
			"type":      "string",
			"minLength": 2,
			"maxLength": 100,
		},
		"model": map[string]interface{}{
			"type":      "string",
			"minLength": 1,
			"maxLength": 100,
		},
		"trim": map[string]interface{}{
			"type":      "string",
			"maxLength": 100,
		},
		"color": map[string]interface{}{
			"type": "string",
			// TODO: Enumeration?
			"maxLength": 100,
		},
		"latitude": map[string]interface{}{
			"type":    "number",
			"minimum": -360,
			"maximum": 360,
		},
		"longitude": map[string]interface{}{
			"type":    "number",
			"minimum": -360,
			"maximum": 360,
		},
		"recaptcha": map[string]interface{}{
			"type": "string",
		},
		"cloudinaryPublicId": map[string]interface{}{
			"type": "string",
		},
	},
	"required": []string{
		"licenseState",
		"licensePlate",
		"year",
		"make",
		"model",
		"color",
		"latitude",
		"longitude",
		"recaptcha",
	},
}

func init() {
	var err error
	postCarsRequestValidator, err = gojsonschema.NewSchema(
		gojsonschema.NewGoLoader(postCarsRequestSchema))
	if err != nil {
		log.Fatal("Error loading POST Cars request schema:", err)
	}
}

type postCarsRequest struct {
	Year               int             `json:"year"`
	Make               string          `json:"make"`
	Model              string          `json:"model"`
	Trim               string          `json:"trim"`
	Color              string          `json:"color"`
	LicenseState       string          `json:"licenseState"`
	LicensePlate       string          `json:"licensePlate"`
	Latitude           decimal.Decimal `json:"latitude"`
	Longitude          decimal.Decimal `json:"longitude"`
	Recaptcha          string          `json:"recaptcha"`
	CloudinaryPublicID string          `json:"cloudinaryPublicId"`
	remoteIP           string
}

func (req postCarsRequest) RecaptchaResponse() string {
	return req.Recaptcha
}

func (req postCarsRequest) RemoteIP() string {
	return req.remoteIP
}

type postCarsResponse struct {
	MapBlockID int `json:"mapBlockId"`
}

func postCarsEndpoint(persistence services.Persistence) endpoint.Endpoint {
	return func(_ context.Context, request interface{}) (interface{}, error) {
		r := request.(postCarsRequest)

		block, err := persistence.GetMapBlock(r.Latitude, r.Longitude)
		if err != nil {
			return nil, encoders.NewJSONError(
				errors.WithMessage(err, "error getting map block"),
				http.StatusInternalServerError)
		}
		// TODO: Do these in a transaction so it can be rolled back in case there's a
		// duplicate key constraint inserting the car.
		if block == nil {
			if err := persistence.InsertMapBlock(r.Latitude, r.Longitude); err != nil {
				return nil, encoders.NewJSONError(
					errors.WithMessage(err, "error inserting map block"),
					http.StatusInternalServerError)
			}
			block, err = persistence.GetMapBlock(r.Latitude, r.Longitude)
			if err != nil {
				return nil, encoders.NewJSONError(
					errors.WithMessage(err, "error getting map block"),
					http.StatusInternalServerError)
			}
		}
		err = persistence.InsertCar(
			r.LicenseState,
			r.LicensePlate,
			block.ID,
			r.Year,
			r.Make,
			r.Model,
			r.Trim,
			r.Color,
			r.CloudinaryPublicID)
		if err != nil {
			// TODO: Handle duplicate key.
			return nil, encoders.NewJSONError(
				errors.WithMessage(err, "error inserting car"),
				http.StatusInternalServerError)
		}

		return postCarsResponse{MapBlockID: block.ID}, nil
	}
}

// maxBodyBytes is the maximum number of bytes that are read
// from the HTTP POST body into memory.
const maxBodyBytes = 5 * 1024 * 1024

func postCarsDecoder(_ context.Context, r *http.Request) (interface{}, error) {
	defer r.Body.Close()

	body, err := ioutil.ReadAll(io.LimitReader(r.Body, maxBodyBytes))
	if err != nil {
		return nil, encoders.NewJSONError(
			errors.WithMessage(err, "error reading body"),
			http.StatusInternalServerError)
	}

	result, err := postCarsRequestValidator.Validate(
		gojsonschema.NewBytesLoader(body))
	if err != nil {
		return nil, encoders.NewJSONError(
			errors.WithMessage(err, "error validating JSON body"),
			http.StatusInternalServerError)
	}
	if !result.Valid() {
		return nil, encoders.NewJSONError(
			// TODO: Is there a good way to display multiple errors?
			errors.New(result.Errors()[0].String()),
			http.StatusBadRequest)
	}

	var req postCarsRequest
	err = json.Unmarshal(body, &req)
	if err != nil {
		return nil, encoders.NewJSONError(
			errors.WithMessage(err, "error unmarshalling JSON body"),
			http.StatusInternalServerError)
	}
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return nil, encoders.NewJSONError(
			errors.WithMessage(err, "failed to get remote IP"),
			http.StatusInternalServerError)
	}
	req.remoteIP = ip
	return req, nil
}

func PostCarsHandler(persistence services.Persistence) http.Handler {
	return httptransport.NewServer(
		middlewares.RecaptchaValidator()(postCarsEndpoint(persistence)),
		postCarsDecoder,
		encoders.JSONResponseEncoder,
	)
}
