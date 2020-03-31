package images

import (
	"context"
	"crypto/sha1"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"path"
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

func (svc Service) deliverySignature(img Image, transform string) string {
	sig := img.Path(transform) + svc.cloudinarySecret
	hash := sha1.Sum([]byte(sig))
	return fmt.Sprintf("s--%s--", base64.URLEncoding.EncodeToString(hash[:])[:8])
}

type Image struct {
	PublicID string
	Format   string
}

func (img Image) Empty() bool {
	return img.PublicID == "" || img.Format == ""
}

func (img Image) Path(transform string) string {
	return path.Join(
		transform,
		fmt.Sprintf("%s.%s", img.PublicID, img.Format))
}

func (svc Service) URL(img Image, transform string) *url.URL {
	if img.Empty() {
		return new(url.URL)
	}

	return &url.URL{
		Scheme: "https",
		Host:   "res.cloudinary.com",
		Path: path.Join(
			"dawfgqsur",
			"image",
			"authenticated",
			svc.deliverySignature(img, transform),
			img.Path(transform)),
	}
}

const insertImageQuery = `
INSERT INTO images (public_id, format)
VALUES ($1, $2)
ON CONFLICT DO NOTHING
`

func (svc Service) InsertImage(publicID, format string) error {
	_, err := svc.db.Exec(insertImageQuery, publicID, format)
	return err
}

const updateImageQuery = `
UPDATE images
SET
	status = $2,
	updated = NOW()
WHERE public_id = $1
`

func (svc Service) UpdateImage(publicID, status string) error {
	_, err := svc.db.Exec(updateImageQuery, publicID, status)
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
	Format           string `json:"format"`
	ModerationStatus string `json:"moderation_status"`
}

type postNotificationResponse struct{}

func postNotificationEndpoint(svc Service) endpoint.Endpoint {
	logErr := func(err error) {
		log.Printf("[postNotificationEndpoint] ERROR: %s: ", err)
	}
	return func(_ context.Context, request interface{}) (interface{}, error) {
		r := request.(postNotificationRequest)
		var err error
		switch r.NotificationType {
		case "upload":
			err = errors.WithMessage(
				svc.InsertImage(r.PublicID, r.Format),
				"error inserting image")
		case "moderation":
			err = errors.WithMessage(
				svc.UpdateImage(r.PublicID, r.ModerationStatus),
				"error updating image")
		}

		if err != nil {
			logErr(err)
			return nil, encoders.NewJSONError(
				errors.New("error handling notification"),
				http.StatusInternalServerError)
		}

		// If the notification type doesn't match anything in our switch, just return OK.
		return "", nil
	}
}

func postNotificationDecoder(svc Service) httptransport.DecodeRequestFunc {
	logErr := func(err error) {
		log.Printf("[postNotificationDecoder] ERROR: %s", err)
	}
	return func(_ context.Context, r *http.Request) (interface{}, error) {
		defer r.Body.Close()
		// Limit the number of bytes of the HTTP POST body read into memory to 5MiB.
		body, err := ioutil.ReadAll(io.LimitReader(r.Body, 5*1024*1024))
		if err != nil {
			msg := "failed to read HTTP body"
			logErr(errors.WithMessage(err, msg))
			return nil, encoders.NewJSONError(
				errors.New(msg),
				http.StatusInternalServerError)
		}

		// Validate the Cloudinary notification signature.
		timestamp := r.Header.Get("x-cld-timestamp")
		expectedSig := svc.NotificationSignature(string(body), timestamp)
		actualSig := r.Header.Get("x-cld-signature")
		if expectedSig != actualSig {
			msg := "signature does not match expected"
			logErr(errors.New(msg))
			return nil, encoders.NewJSONError(
				errors.New(msg),
				http.StatusUnauthorized)
		}

		var req postNotificationRequest
		err = json.Unmarshal(body, &req)
		if err != nil {
			msg := "failed to unmarshal JSON body"
			logErr(errors.WithMessage(err, msg))
			return nil, encoders.NewJSONError(
				errors.New(msg),
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
