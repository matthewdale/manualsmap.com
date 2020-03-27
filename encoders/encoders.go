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

func NewJSONError(err error, statusCode int) error {
	if err == nil {
		return nil
	}
	return &JSONError{err, statusCode}
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
