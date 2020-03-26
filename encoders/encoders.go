package encoders

import (
	"context"
	"encoding/json"
	"net/http"

	httptransport "github.com/go-kit/kit/transport/http"
)

func JSONResponseEncoder(_ context.Context, writer http.ResponseWriter, response interface{}) error {
	if res, ok := response.(httptransport.StatusCoder); ok {
		if code := res.StatusCode(); code > 0 {
			writer.WriteHeader(res.StatusCode())
		}
	}
	writer.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(writer).Encode(response)
}

type JSONError struct {
	error
	statusCode int
}

func NewJSONError(err error, statusCode int) JSONError {
	return JSONError{err, statusCode}
}

func (err JSONError) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Err string `json:"err"`
	}{
		Err: err.Error(),
	})
}

func (err JSONError) StatusCode() int {
	if err.statusCode < 100 || err.statusCode >= 600 {
		return http.StatusInternalServerError
	}
	return err.statusCode
}

// func JSONErrorsEndpoint(e endpoint.Endpoint) endpoint.Endpoint {
// 	return func(ctx context.Context, request interface{}) (interface{}, error) {
// 		resp, err := e(ctx, request)
// 		if err != nil {
// 			return nil, jsonError{err, 0}
// 		}
// 		return resp, nil
// 	}
// }

// func JSONErrorsDecodeRequestFunc(d httptransport.DecodeRequestFunc) httptransport.DecodeRequestFunc {
// 	return func(ctx context.Context, r *http.Request) (interface{}, error) {
// 		resp, err := d(ctx, r)
// 		if err != nil {
// 			return nil, jsonError{err, 0}
// 		}
// 		return resp, nil
// 	}
// }
