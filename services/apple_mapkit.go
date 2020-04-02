package services

import (
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/pem"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/pkg/errors"
)

type AppleMapkit struct {
	teamID     string
	keyID      string
	privateKey *ecdsa.PrivateKey
	origin     string
}

func NewAppleMapkit(
	teamID,
	keyID string,
	pemPrivateKey []byte,
	origin string,
) (AppleMapkit, error) {
	block, _ := pem.Decode(pemPrivateKey)
	if block == nil || block.Type != "PRIVATE KEY" {
		return AppleMapkit{}, errors.New("invalid private key block")
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return AppleMapkit{}, errors.WithMessage(err, "error parsing private key")
	}

	return AppleMapkit{
		teamID:     teamID,
		keyID:      keyID,
		privateKey: key.(*ecdsa.PrivateKey),
	}, nil
}

func (svc AppleMapkit) GetToken() (string, error) {
	claims := jwt.MapClaims{
		"iss": svc.teamID,
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(1 * time.Hour).Unix(),
	}
	if svc.origin != "" {
		claims["origin"] = svc.origin
	}
	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	token.Header["kid"] = svc.keyID

	tokenString, err := token.SignedString(svc.privateKey)
	if err != nil {
		return "", errors.WithMessage(err, "error generating signed JWT")
	}
	return tokenString, nil
}
