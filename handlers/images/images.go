package images

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"sort"
	"strings"

	"github.com/go-kit/kit/endpoint"
	httptransport "github.com/go-kit/kit/transport/http"
	"github.com/matthewdale/manualsmap.com/encoders"
	"github.com/pkg/errors"
)

type Service struct {
	cloudinarySecret string
}

func NewService(cloudinarySecret string) Service {
	return Service{cloudinarySecret: cloudinarySecret}
}

// encode encodes parameters in the Cloudinary signature
// string format.
func encode(parameters map[string]string) string {
	var buf strings.Builder

	// Collect and sort the keys in ascending order.
	keys := make([]string, 0, len(parameters))
	for k := range parameters {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Build the encoded signature string like
	// key1=value1&key2=value2&...
	for _, key := range keys {
		val := parameters[key]
		if buf.Len() > 0 {
			buf.WriteByte('&')
		}
		buf.WriteString(key)
		buf.WriteByte('=')
		buf.WriteString(val)
	}

	return buf.String()
}

func (svc Service) Signature(parameters map[string]string) string {
	if parameters == nil {
		return ""
	}
	sig := encode(parameters) + svc.cloudinarySecret
	hash := sha1.Sum([]byte(sig))
	return hex.EncodeToString(hash[:])
}

type postSignatureRequest struct {
	Parameters map[string]string `json:"parameters"`
}

type postSignatureResponse struct {
	Signature string `json:"signature"`
}

func postSignatureEndpoint(svc Service) endpoint.Endpoint {
	return func(_ context.Context, request interface{}) (interface{}, error) {
		parameters := request.(postSignatureRequest).Parameters
		// Only allow uploads using the "manualsmap_com" preset. If any
		// other upload preset is set, override it, which will cause a
		// signature failure on the client side and prevent the upload.
		if _, ok := parameters["upload_preset"]; ok {
			parameters["upload_preset"] = "manualsmap_com"
		}
		signature := svc.Signature(parameters)

		return postSignatureResponse{Signature: signature}, nil
	}
}

// maxBodyBytes is the maximum number of bytes that are read
// from the HTTP POST body into memory.
const maxBodyBytes = 1 * 1024 * 1024

func postCarsDecoder(_ context.Context, r *http.Request) (interface{}, error) {
	defer r.Body.Close()

	var req postSignatureRequest
	err := json.NewDecoder(io.LimitReader(r.Body, maxBodyBytes)).Decode(&req)
	if err != nil {
		return nil, encoders.NewJSONError(
			errors.WithMessage(err, "error unmarshalling JSON body"),
			http.StatusInternalServerError)
	}
	return req, nil
}

func PostSignatureHandler(svc Service) http.Handler {
	return httptransport.NewServer(
		postSignatureEndpoint(svc),
		postCarsDecoder,
		encoders.JSONResponseEncoder,
	)
}
