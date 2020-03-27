package images

import (
	"context"
	"crypto/sha1"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"sort"
	"strings"

	"github.com/go-kit/kit/endpoint"
	httptransport "github.com/go-kit/kit/transport/http"
	"github.com/matthewdale/manualsmap.com/encoders"
	"github.com/pkg/errors"
)

type Service struct {
	db               *sql.DB
	cloudinarySecret string
}

func NewService(db *sql.DB, cloudinarySecret string) Service {
	return Service{db: db, cloudinarySecret: cloudinarySecret}
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

func (svc Service) UploadSignature(parameters map[string]string) string {
	if parameters == nil {
		return ""
	}
	sig := encode(parameters) + svc.cloudinarySecret
	hash := sha1.Sum([]byte(sig))
	return hex.EncodeToString(hash[:])
}

func (svc Service) NotificationSignature(body string, timestamp string) string {
	sig := body + timestamp + svc.cloudinarySecret
	hash := sha1.Sum([]byte(sig))
	return hex.EncodeToString(hash[:])
}

const insertImageQuery = `
INSERT INTO images (public_id, format, version)
VALUES ($1, $2, $3)
ON CONFLICT DO NOTHING
`

func (svc Service) InsertImage(publicID, format string, version int) error {
	_, err := svc.db.Exec(insertImageQuery, publicID, format, version)
	return err
}

const updateImageQuery = `
UPDATE images
SET
	version = $2,
	status = $3,
	updated = NOW()
WHERE public_id = $1
`

func (svc Service) UpdateImage(publicID string, version int, status string) error {
	_, err := svc.db.Exec(updateImageQuery, publicID, version, status)
	return err
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
		signature := svc.UploadSignature(parameters)

		return postSignatureResponse{Signature: signature}, nil
	}
}

func postSignatureDecoder(_ context.Context, r *http.Request) (interface{}, error) {
	defer r.Body.Close()

	var req postSignatureRequest
	// Limit the number of bytes of the HTTP POST body read into memory to 1MiB.
	err := json.NewDecoder(io.LimitReader(r.Body, 1*1024*1024)).Decode(&req)
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
		postSignatureDecoder,
		encoders.JSONResponseEncoder,
	)
}

type postNotificationRequest struct {
	NotificationType string `json:"notification_type"`
	PublicID         string `json:"public_id"`
	Version          int    `json:"version"`
	Format           string `json:"format"`
	ModerationStatus string `json:"moderation_status"`
}

type postNotificationResponse struct{}

func postNotificationEndpoint(svc Service) endpoint.Endpoint {
	return func(_ context.Context, request interface{}) (interface{}, error) {
		r := request.(postNotificationRequest)
		switch r.NotificationType {
		case "upload":
			err := svc.InsertImage(r.PublicID, r.Format, r.Version)
			return "", encoders.NewJSONError(
				errors.WithMessage(err, "error inserting image"),
				http.StatusInternalServerError)
		case "moderation":
			err := svc.UpdateImage(r.PublicID, r.Version, r.ModerationStatus)
			return "", encoders.NewJSONError(
				errors.WithMessage(err, "error inserting image"),
				http.StatusInternalServerError)
		}

		// If the notification type doesn't match anything in our switch, just return OK.
		return "", nil
	}
}

func postNotificationDecoder(svc Service) httptransport.DecodeRequestFunc {
	return func(_ context.Context, r *http.Request) (interface{}, error) {
		defer r.Body.Close()
		// Limit the number of bytes of the HTTP POST body read into memory to 5MiB.
		body, err := ioutil.ReadAll(io.LimitReader(r.Body, 5*1024*1024))
		if err != nil {
			return nil, encoders.NewJSONError(
				errors.WithMessage(err, "error reading body"),
				http.StatusInternalServerError)
		}

		// Validate the Cloudinary notification signature.
		timestamp := r.Header.Get("x-cld-timestamp")
		expectedSig := svc.NotificationSignature(string(body), timestamp)
		actualSig := r.Header.Get("x-cld-signature")
		if expectedSig != actualSig {
			return nil, encoders.NewJSONError(
				errors.New("signature does not match expected"),
				http.StatusUnauthorized)
		}

		var req postNotificationRequest
		err = json.Unmarshal(body, &req)
		if err != nil {
			return nil, encoders.NewJSONError(
				errors.WithMessage(err, "error unmarshalling JSON body"),
				http.StatusInternalServerError)
		}
		return req, nil
	}
}

func PostNotificationHandler(svc Service) http.Handler {
	return httptransport.NewServer(
		postNotificationEndpoint(svc),
		postNotificationDecoder(svc),
		func(_ context.Context, writer http.ResponseWriter, _ interface{}) error {
			writer.WriteHeader(http.StatusOK)
			return nil
		},
	)
}
