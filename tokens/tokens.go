package tokens

import (
	"context"
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/pem"
	"net/http"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/go-kit/kit/endpoint"
	httptransport "github.com/go-kit/kit/transport/http"
	"github.com/pkg/errors"

	"github.com/matthewdale/manualsmap.com/encoders"
)

type Service struct {
	teamID     string
	keyID      string
	privateKey *ecdsa.PrivateKey
}

func NewService(teamID string, keyID string, pemKey []byte) (Service, error) {
	block, _ := pem.Decode(pemKey)
	if block == nil || block.Type != "PRIVATE KEY" {
		return Service{}, errors.New("invalid private key block")
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return Service{}, errors.WithMessage(err, "error parsing private key")
	}

	return Service{
		teamID:     teamID,
		keyID:      keyID,
		privateKey: key.(*ecdsa.PrivateKey),
	}, nil
}

func (svc Service) GetToken() (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodES256, jwt.MapClaims{
		"iss": svc.teamID,
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(1 * time.Hour).Unix(),
		// "origin": origin,
	})
	token.Header["kid"] = svc.keyID

	tokenString, err := token.SignedString(svc.privateKey)
	if err != nil {
		return "", errors.WithMessage(err, "error generating signed JWT")
	}
	return tokenString, nil
}

type getResponse struct {
	Token      string `json:"token,omitempty"`
	Err        string `json:"err,omitempty"`
	statusCode int
}

func (res getResponse) StatusCode() int {
	return res.statusCode
}

func getEndpoint(svc Service) endpoint.Endpoint {
	return func(_ context.Context, request interface{}) (interface{}, error) {
		token, err := svc.GetToken()
		if err != nil {
			return getResponse{Err: err.Error(), statusCode: http.StatusInternalServerError}, nil
		}
		return getResponse{Token: token}, nil
	}
}

func GetHandler(svc Service) http.Handler {
	return httptransport.NewServer(
		getEndpoint(svc),
		func(_ context.Context, r *http.Request) (interface{}, error) { return nil, nil },
		encoders.JSONResponseEncoder,
	)
}