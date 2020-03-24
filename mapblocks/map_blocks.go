package mapblocks

import (
	"context"
	"database/sql"
	"net/http"

	"github.com/go-kit/kit/endpoint"
	httptransport "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/schema"
	"github.com/pkg/errors"

	"github.com/matthewdale/manualsmap.com/encoders"
)

type Service struct {
	db *sql.DB
}

func NewService(db *sql.DB) Service {
	return Service{db: db}
}

type MapBlock struct {
	ID        int     `json:"id"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// TODO: Adjust limit.
const getMapBlocksQuery = `
SELECT
	id, latitude, longitude
FROM map_blocks
WHERE
	latitude BETWEEN TRUNC($1, 2)-0.1 AND TRUNC($2, 2)+0.1
	AND longitude BETWEEN TRUNC($3, 2)-0.1 AND TRUNC($4, 2)+0.1
LIMIT 100
`

func (svc Service) GetMapBlocks(
	minLatitude,
	minLongitude,
	maxLatitude,
	maxLongitude float64,
) ([]MapBlock, error) {
	rows, err := svc.db.Query(
		getMapBlocksQuery,
		minLatitude,
		maxLatitude,
		minLongitude,
		maxLongitude)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to read map blocks")
	}
	blocks := make([]MapBlock, 0, 10)

	for rows.Next() {
		var block MapBlock
		err := rows.Scan(
			&block.ID,
			&block.Latitude,
			&block.Longitude)
		if err != nil {
			return nil, errors.WithMessage(err, "failed to scan map block row into struct")
		}
		blocks = append(blocks, block)
	}
	return blocks, nil
}

const getMapBlockQuery = `
SELECT
	id, latitude, longitude
FROM map_blocks
WHERE
	latitude = TRUNC($1, 2)
	AND longitude = TRUNC($2, 2)
`

func (svc Service) GetMapBlock(latitude, longitude float64) (*MapBlock, error) {
	var block MapBlock
	err := svc.db.QueryRow(getMapBlockQuery, latitude, longitude).Scan(
		&block.ID,
		&block.Longitude,
		&block.Latitude,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, errors.WithMessage(err, "failed to read map blocks")
	}
	return &block, nil
}

const insertMapBlockQuery = `
INSERT INTO map_blocks (latitude, longitude)
VALUES (TRUNC($1, 2), TRUNC($2, 2))
ON CONFLICT DO NOTHING
`

func (svc Service) InsertMapBlock(latitude, longitude float64) error {
	_, err := svc.db.Exec(insertMapBlockQuery, latitude, longitude)
	return err
}

type getRequest struct {
	MinLatitude  float64 `schema:"min_latitude"`
	MinLongitude float64 `schema:"min_longitude"`
	MaxLatitude  float64 `schema:"max_latitude"`
	MaxLongitude float64 `schema:"max_longitude"`
}

type getResponse struct {
	MapBlocks  []MapBlock `json:"mapBlocks"`
	Err        string     `json:"err,omitempty"`
	statusCode int
}

func (res getResponse) StatusCode() int {
	return res.statusCode
}

func getEndpoint(svc Service) endpoint.Endpoint {
	return func(_ context.Context, request interface{}) (interface{}, error) {
		r := request.(getRequest)
		mapBlocks, err := svc.GetMapBlocks(
			r.MinLatitude,
			r.MinLongitude,
			r.MaxLatitude,
			r.MaxLongitude)
		if err != nil {
			return getResponse{Err: err.Error(), statusCode: http.StatusInternalServerError}, nil
		}
		return getResponse{MapBlocks: mapBlocks}, nil
	}
}

func getDecode(_ context.Context, r *http.Request) (interface{}, error) {
	var req getRequest
	decoder := schema.NewDecoder()
	decoder.Decode(&req, r.URL.Query())
	return req, nil
}

func GetHandler(svc Service) http.Handler {
	return httptransport.NewServer(
		getEndpoint(svc),
		getDecode,
		encoders.JSONResponseEncoder,
	)
}
