package mapblocks

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/dpapathanasiou/go-recaptcha"
	"github.com/go-kit/kit/endpoint"
	httptransport "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/xeipuuv/gojsonschema"
	"golang.org/x/crypto/scrypt"

	"github.com/matthewdale/manualsmap.com/encoders"
	"github.com/matthewdale/manualsmap.com/handlers/images"
)

type Car struct {
	Year  int
	Brand string
	Model string
	Trim  string
	Color string
	Image images.Image
}

// TODO: Fetch additional columns, order by created descending,
// and paginate.
const getCarsQuery = `
SELECT
	c.year,
	c.brand,
	c.model,
	c.trim,
	c.color,
	i.public_id,
	i.format
FROM cars c
LEFT JOIN images i ON
	i.public_id = c.images_public_id
	AND i.status = 'approved'
WHERE c.map_block_id = $1
ORDER BY c.created DESC
`

func (svc Service) GetCars(mapBlockID int) ([]Car, error) {
	rows, err := svc.db.Query(getCarsQuery, mapBlockID)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to read cars")
	}
	cars := make([]Car, 0, 10)
	for rows.Next() {
		var car Car
		var publicID sql.NullString
		var format sql.NullString
		err := rows.Scan(
			&car.Year,
			&car.Brand,
			&car.Model,
			&car.Trim,
			&car.Color,
			&publicID,
			&format)
		if err != nil {
			return nil, errors.WithMessage(err, "failed to scan car row into struct")
		}
		// TODO: If the image is awaiting moderation, add an "awaiting moderation" image.
		// TODO: If there is no image, add a stock photo.
		if publicID.Valid && format.Valid {
			car.Image.PublicID = publicID.String
			car.Image.Format = format.String
		}
		cars = append(cars, car)
	}
	return cars, nil
}

var nonAlphanumeric = regexp.MustCompile(`[^\w\d]`)

func (svc Service) licenseHash(licenseState, licensePlate string) (string, error) {
	license := []byte(fmt.Sprintf("%s-%s",
		strings.ToUpper(strings.TrimSpace(licenseState)),
		strings.ToUpper(nonAlphanumeric.ReplaceAllString(licensePlate, ""))))
	hash, err := scrypt.Key(license, svc.salt, 1<<15, 8, 1, 32)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(hash), nil
}

const insertCarQuery = `
INSERT INTO cars(
	license_hash,
	map_block_id,
	year,
	brand,
	model,
	trim,
	color,
	images_public_id
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
`

func (svc Service) InsertCar(
	licenseHash string,
	mapBlockID,
	year int,
	brand,
	model,
	trim,
	color,
	imagePublicID string,
) error {
	_, err := svc.db.Exec(
		insertCarQuery,
		licenseHash,
		mapBlockID,
		year,
		strings.TrimSpace(brand),
		strings.TrimSpace(model),
		strings.TrimSpace(trim),
		strings.ToLower(strings.TrimSpace(color)),
		strings.TrimSpace(imagePublicID))
	if err != nil {
		return errors.WithMessage(err, "failed to insert car")
	}
	return nil
}

type getCarsRequest struct {
	mapBlockID int
}

type carResponse struct {
	Year         int    `json:"year"`
	Brand        string `json:"brand"`
	Model        string `json:"model"`
	Trim         string `json:"trim"`
	Color        string `json:"color"`
	ImageURL     string `json:"imageUrl"`
	ThumbnailURL string `json:"thumbnailUrl"`
}

type getCarsResponse struct {
	Cars []carResponse `json:"cars"`
}

func getCarsEndpoint(svc Service, imagesSvc images.Service) endpoint.Endpoint {
	return func(_ context.Context, request interface{}) (interface{}, error) {
		cars, err := svc.GetCars(request.(getCarsRequest).mapBlockID)
		if err != nil {
			return nil, encoders.NewJSONError(
				errors.WithMessage(err, "error getting car"),
				http.StatusInternalServerError)
		}

		carResponses := make([]carResponse, 0, len(cars))
		for _, car := range cars {
			carResponses = append(carResponses, carResponse{
				Year:         car.Year,
				Brand:        car.Brand,
				Model:        car.Model,
				Trim:         car.Trim,
				Color:        car.Color,
				ImageURL:     imagesSvc.URL(car.Image, "").String(),
				ThumbnailURL: imagesSvc.URL(car.Image, "c_limit,w_300").String(),
			})
		}
		return getCarsResponse{Cars: carResponses}, nil
	}
}

func GetCarsHandler(svc Service, imagesSvc images.Service) http.Handler {
	return httptransport.NewServer(
		getCarsEndpoint(svc, imagesSvc),
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

type postCarsRequest struct {
	Year               int     `json:"year"`
	Brand              string  `json:"brand"`
	Model              string  `json:"model"`
	Trim               string  `json:"trim"`
	Color              string  `json:"color"`
	LicenseState       string  `json:"licenseState"`
	LicensePlate       string  `json:"licensePlate"`
	Latitude           float64 `json:"latitude"`
	Longitude          float64 `json:"longitude"`
	Recaptcha          string  `json:"recaptcha"`
	CloudinaryPublicID string  `json:"cloudinaryPublicId"`
	remoteIP           string
}

type postCarsResponse struct {
	LicenseHash string `json:"license_hash"`
}

var postCarsRequestSchema *gojsonschema.Schema

func init() {
	var err error
	postCarsRequestSchema, err = gojsonschema.NewSchema(gojsonschema.NewGoLoader(map[string]interface{}{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"type":    "object",
		"properties": map[string]interface{}{
			"year": map[string]interface{}{
				"type":    "number",
				"minimum": 1900,
				"maximum": 2100,
			},
			"brand": map[string]interface{}{
				"type":      "string",
				"minLength": 2,
			},
			"model": map[string]interface{}{
				"type":      "string",
				"minLength": 1,
			},
			"trim": map[string]interface{}{
				"type": "string",
			},
			"color": map[string]interface{}{
				"type": "string",
			},
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
		"required": []string{"year", "brand", "model", "color", "licenseState", "licensePlate", "latitude", "longitude", "recaptcha"},
	}))
	if err != nil {
		log.Fatal("Error loading POST Cars request schema:", err)
	}
}

func postCarsEndpoint(svc Service) endpoint.Endpoint {
	return func(_ context.Context, request interface{}) (interface{}, error) {
		r := request.(postCarsRequest)

		// reCAPTCHA form validation.
		valid, err := recaptcha.Confirm(r.remoteIP, r.Recaptcha)
		if err != nil {
			return nil, encoders.NewJSONError(
				errors.WithMessage(err, "reCAPTCHA server error"),
				http.StatusInternalServerError)
		}
		if !valid {
			return nil, encoders.NewJSONError(
				errors.New("reCAPTCHA validation failed"),
				http.StatusForbidden)
		}

		hash, err := svc.licenseHash(r.LicenseState, r.LicensePlate)
		if err != nil {
			return nil, encoders.NewJSONError(
				errors.WithMessage(err, "error generating license hash"),
				http.StatusInternalServerError)
		}
		block, err := svc.GetMapBlock(r.Latitude, r.Longitude)
		if err != nil {
			return nil, encoders.NewJSONError(
				errors.WithMessage(err, "error getting map block"),
				http.StatusInternalServerError)
		}
		// TODO: Do these in a transaction so it can be rolled back in case there's a
		// duplicate key constraint inserting the car.
		if block == nil {
			if err := svc.InsertMapBlock(r.Latitude, r.Longitude); err != nil {
				return nil, encoders.NewJSONError(
					errors.WithMessage(err, "error inserting map block"),
					http.StatusInternalServerError)
			}
			block, err = svc.GetMapBlock(r.Latitude, r.Longitude)
			if err != nil {
				return nil, encoders.NewJSONError(
					errors.WithMessage(err, "error getting map block"),
					http.StatusInternalServerError)
			}
		}
		err = svc.InsertCar(
			hash,
			block.ID,
			r.Year,
			r.Brand,
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
		return postCarsResponse{LicenseHash: hash}, nil
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

	result, err := postCarsRequestSchema.Validate(gojsonschema.NewBytesLoader(body))
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

func PostCarsHandler(svc Service) http.Handler {
	return httptransport.NewServer(
		postCarsEndpoint(svc),
		postCarsDecoder,
		encoders.JSONResponseEncoder,
	)
}
