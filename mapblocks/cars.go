package mapblocks

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-kit/kit/endpoint"
	httptransport "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"

	"github.com/matthewdale/manualsmap.com/encoders"
)

type Car struct {
	Year     int    `json:"year"`
	Brand    string `json:"brand"`
	Model    string `json:"model"`
	Trim     string `json:"trim"`
	Color    string `json:"color"`
	ImageURL string `json:"imageUrl"`
}

const getCarsQuery = `
SELECT
	year, brand, model, trim, color, image_url
FROM cars
WHERE map_block_id = $1
`

func (svc Service) GetCars(mapBlockID int) ([]Car, error) {
	rows, err := svc.db.Query(getCarsQuery, mapBlockID)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to read cars")
	}
	cars := make([]Car, 0, 10)
	for rows.Next() {
		var car Car
		err := rows.Scan(
			&car.Year,
			&car.Brand,
			&car.Model,
			&car.Trim,
			&car.Color,
			&car.ImageURL)
		if err != nil {
			return nil, errors.WithMessage(err, "failed to scan car row into struct")
		}
		cars = append(cars, car)
	}
	return cars, nil
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
	image_url
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
`

func (svc Service) InsertCar(
	car Car,
	licenseHash string,
	mapBlockID int,
) error {
	_, err := svc.db.Exec(
		insertCarQuery,
		licenseHash,
		mapBlockID,
		car.Year,
		car.Brand,
		car.Model,
		car.Trim,
		strings.ToLower(car.Color),
		car.ImageURL)
	if err != nil {
		return errors.WithMessage(err, "failed to insert car")
	}
	return nil
}

type getCarsRequest struct {
	mapBlockID int
}

type getCarsResponse struct {
	Cars       []Car  `json:"cars"`
	Err        string `json:"err,omitempty"`
	statusCode int
}

func (res getCarsResponse) StatusCode() int {
	return res.statusCode
}

func getCarsEndpoint(svc Service) endpoint.Endpoint {
	return func(_ context.Context, request interface{}) (interface{}, error) {
		cars, err := svc.GetCars(request.(getCarsRequest).mapBlockID)
		if err != nil {
			return getCarsResponse{Err: err.Error(), statusCode: http.StatusInternalServerError}, nil
		}
		return getCarsResponse{Cars: cars}, nil
	}
}

func GetCarsHandler(svc Service) http.Handler {
	return httptransport.NewServer(
		getCarsEndpoint(svc),
		func(_ context.Context, r *http.Request) (interface{}, error) {
			vars := mux.Vars(r)
			id, ok := vars["id"]
			if !ok {
				return nil, errors.New("invalid request, missing {id}")
			}
			mapBlockID, err := strconv.Atoi(id)
			if err != nil {
				return nil, errors.WithMessage(err, "invalid {id} format")
			}
			return getCarsRequest{mapBlockID: mapBlockID}, nil
		},
		encoders.JSONResponseEncoder,
	)
}

type postCarsRequest struct {
	Car          Car     `json:"car"`
	LicenseState string  `json:"licenseState"`
	LicensePlate string  `json:"licensePlate"`
	Latitude     float64 `json:"latitude"`
	Longitude    float64 `json:"longitude"`
}

type postCarsResponse struct {
	LicenseHash string `json:"license_hash"`
}

func licenseHash(licenseState, licensePlate string) (string, error) {
	// TODO: Salt + hash input.
	// TODO: Validate state enumeration.
	return strings.ToLower(licenseState) + strings.ToUpper(licensePlate), nil
}

func postCarsEndpoint(svc Service) endpoint.Endpoint {
	return func(_ context.Context, request interface{}) (interface{}, error) {
		// TODO: Replace with errors that implement json.Marshaler and
		// httptransport.StatusCoder
		r := request.(postCarsRequest)
		hash, err := licenseHash(r.LicensePlate, r.LicenseState)
		if err != nil {
			return nil, err
		}
		block, err := svc.GetMapBlock(r.Latitude, r.Longitude)
		if err != nil {
			return nil, err
		}
		// TODO: Do these in a transaction so it can be rolled back in case there's a
		// duplicate key constraint inserting the car.
		if block == nil {
			if err := svc.InsertMapBlock(r.Latitude, r.Longitude); err != nil {
				return nil, err
			}
			block, err = svc.GetMapBlock(r.Latitude, r.Longitude)
			if err != nil {
				return nil, err
			}
		}
		err = svc.InsertCar(r.Car, hash, block.ID)
		if err != nil {
			// TODO: Handle duplicate key.
			return nil, err
		}
		return postCarsResponse{LicenseHash: hash}, nil
	}
}

func PostCarsHandler(svc Service) http.Handler {
	return httptransport.NewServer(
		postCarsEndpoint(svc),
		func(_ context.Context, r *http.Request) (interface{}, error) {
			var req postCarsRequest
			err := json.NewDecoder(r.Body).Decode(&req)
			if err != nil {
				return nil, errors.WithMessage(err, "invalid POST body")
			}
			return req, nil
		},
		encoders.JSONResponseEncoder,
	)
}
