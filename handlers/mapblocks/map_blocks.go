package mapblocks

import (
	"context"
	"net/http"

	"github.com/go-kit/kit/endpoint"
	httptransport "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/schema"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"

	"github.com/matthewdale/manualsmap.com/encoders"
	"github.com/matthewdale/manualsmap.com/services"
)

type getMapBlocksRequest struct {
	MinLatitude  decimal.Decimal `schema:"min_latitude"`
	MinLongitude decimal.Decimal `schema:"min_longitude"`
	MaxLatitude  decimal.Decimal `schema:"max_latitude"`
	MaxLongitude decimal.Decimal `schema:"max_longitude"`
}

type mapBlock struct {
	ID        int             `json:"id"`
	Latitude  decimal.Decimal `json:"latitude"`
	Longitude decimal.Decimal `json:"longitude"`
}

type getMapBlocksResponse struct {
	MapBlocks []mapBlock `json:"mapBlocks"`
}

func getEndpoint(persistence services.Persistence) endpoint.Endpoint {
	return func(_ context.Context, request interface{}) (interface{}, error) {
		r := request.(getMapBlocksRequest)
		mapBlocks, err := persistence.GetMapBlocks(
			r.MinLatitude,
			r.MinLongitude,
			r.MaxLatitude,
			r.MaxLongitude)
		if err != nil {
			return nil, encoders.NewJSONError(
				errors.WithMessage(err, "error getting map block"),
				http.StatusInternalServerError)
		}
		responseBlocks := make([]mapBlock, 0, len(mapBlocks))
		for _, block := range mapBlocks {
			responseBlocks = append(responseBlocks, mapBlock{
				ID:        block.ID,
				Latitude:  block.Latitude,
				Longitude: block.Longitude,
			})
		}
		return getMapBlocksResponse{MapBlocks: responseBlocks}, nil
	}
}

func getDecode(_ context.Context, r *http.Request) (interface{}, error) {
	var req getMapBlocksRequest
	decoder := schema.NewDecoder()
	decoder.Decode(&req, r.URL.Query())
	return req, nil
}

func GetHandler(persistence services.Persistence) http.Handler {
	return httptransport.NewServer(
		getEndpoint(persistence),
		getDecode,
		encoders.JSONResponseEncoder,
	)
}
