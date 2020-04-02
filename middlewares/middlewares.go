package middlewares

import (
	"context"
	"net/http"

	"github.com/dpapathanasiou/go-recaptcha"
	"github.com/go-kit/kit/endpoint"
	"github.com/matthewdale/manualsmap.com/encoders"
	"github.com/pkg/errors"
)

type RecaptchaRequest interface {
	Recaptcha() string
	RemoteIP() string
}

func RecaptchaValidator() endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, request interface{}) (interface{}, error) {
			// Do an unchecked type assertion here. If the type assertion fails,
			// the call will panic, which is OK because it's better than completely
			// skipping request validation.
			r := request.(RecaptchaRequest)
			valid, err := recaptcha.Confirm(r.RemoteIP(), r.Recaptcha())
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
			return next(ctx, request)
		}
	}
}
