package images

import (
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/go-kit/kit/endpoint"
	httptransport "github.com/go-kit/kit/transport/http"
	"github.com/matthewdale/manualsmap.com/encoders"
	"github.com/matthewdale/manualsmap.com/services"
	"github.com/pkg/errors"
)

type postSignatureRequest struct {
	Parameters map[string]string `json:"parameters"`
}

type postSignatureResponse struct {
	Signature string `json:"signature"`
}

func postSignatureEndpoint(cloudinary services.Cloudinary) endpoint.Endpoint {
	return func(_ context.Context, request interface{}) (interface{}, error) {
		parameters := request.(postSignatureRequest).Parameters
		// Only allow uploads using the "manualsmap_com" preset. If any
		// other upload preset is set, override it, which will cause a
		// signature failure on the client side and prevent the upload.
		if _, ok := parameters["upload_preset"]; ok {
			parameters["upload_preset"] = "manualsmap_com"
		}
		signature := cloudinary.UploadSignature(parameters)

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

func PostSignatureHandler(cloudinary services.Cloudinary) http.Handler {
	return httptransport.NewServer(
		// TODO: Require reCAPTCHA validation before generating an upload signature.
		// middlewares.RecaptchaValidator()(postSignatureEndpoint(cloudinary)),
		postSignatureEndpoint(cloudinary),
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

func postNotificationEndpoint(persistence services.Persistence) endpoint.Endpoint {
	logErr := func(err error) {
		log.Printf("[postNotificationEndpoint] ERROR: %s: ", err)
	}
	return func(_ context.Context, request interface{}) (interface{}, error) {
		r := request.(postNotificationRequest)
		var err error
		switch r.NotificationType {
		case "upload":
			err = errors.WithMessage(
				persistence.InsertImage(r.PublicID, r.Format),
				"error inserting image")
		case "moderation":
			err = errors.WithMessage(
				persistence.UpdateImage(r.PublicID, r.ModerationStatus),
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

func postNotificationDecoder(
	cloudinary services.Cloudinary,
) httptransport.DecodeRequestFunc {
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
		expectedSig := cloudinary.NotificationSignature(string(body), timestamp)
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

func PostNotificationHandler(
	persistence services.Persistence,
	cloudinary services.Cloudinary,
) http.Handler {
	return httptransport.NewServer(
		postNotificationEndpoint(persistence),
		postNotificationDecoder(cloudinary),
		func(_ context.Context, writer http.ResponseWriter, _ interface{}) error {
			writer.WriteHeader(http.StatusOK)
			return nil
		},
	)
}
