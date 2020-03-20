package encoders

import (
	"context"
	"encoding/json"
	"net/http"
)

type HTTPCodedResponse interface {
	StatusCode() int
}

func JSONResponseEncoder(_ context.Context, writer http.ResponseWriter, response interface{}) error {
	if res, ok := response.(HTTPCodedResponse); ok {
		if code := res.StatusCode(); code > 0 {
			writer.WriteHeader(res.StatusCode())
		}
	}
	writer.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(writer).Encode(response)
}
