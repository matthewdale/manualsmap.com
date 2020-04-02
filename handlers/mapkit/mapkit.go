// Package mapkit provides HTTP handlers for initializing the
// Apple MapKit JS maps toolkit.
package mapkit

import (
	"context"
	"net/http"

	"github.com/go-kit/kit/endpoint"
	httptransport "github.com/go-kit/kit/transport/http"
	"github.com/pkg/errors"

	"github.com/matthewdale/manualsmap.com/encoders"
	"github.com/matthewdale/manualsmap.com/services"
)

type getTokenResponse struct {
	Token string `json:"token"`
}

func getTokenEndpoint(mapkit services.AppleMapkit) endpoint.Endpoint {
	return func(_ context.Context, request interface{}) (interface{}, error) {
		token, err := mapkit.GetToken()
		if err != nil {
			return nil, encoders.NewJSONError(
				errors.WithMessage(err, "error getting token"),
				http.StatusInternalServerError)
		}
		return getTokenResponse{Token: token}, nil
	}
}

func GetTokenHandler(mapkit services.AppleMapkit) http.Handler {
	return httptransport.NewServer(
		getTokenEndpoint(mapkit),
		func(_ context.Context, r *http.Request) (interface{}, error) { return nil, nil },
		encoders.JSONResponseEncoder,
	)
}
