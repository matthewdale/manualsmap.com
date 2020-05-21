package services

import (
	"database/sql"
	"encoding/base64"
	"fmt"
	"regexp"
	"strings"

	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"golang.org/x/crypto/scrypt"
)

// Persistence is a service that provides persistence for all service data in a
// SQL database.
type Persistence struct {
	db   *sql.DB
	salt []byte
}

// NewPersistence creates a new Persistence service. The salt is used to
// obfuscate the license plate information and must remain the same to detect
// duplicate car entries.
func NewPersistence(db *sql.DB, salt []byte) Persistence {
	return Persistence{db: db, salt: salt}
}

type MapBlock struct {
	ID        int
	Latitude  decimal.Decimal
	Longitude decimal.Decimal
}

var mapBlockSize = decimal.NewFromFloat(0.05)

func segmentCoordinate(coordinate decimal.Decimal) decimal.Decimal {
	return coordinate.Div(mapBlockSize).Truncate(0).Mul(mapBlockSize)
}

// TODO: Adjust limit.
const getMapBlocksQuery = `
SELECT
	id, latitude, longitude
FROM map_blocks
WHERE
	latitude BETWEEN $1 AND $2
	AND longitude BETWEEN $3 AND $4
LIMIT 100
`

var coordinateOvershoot = decimal.NewFromFloat(0.5)

func (svc Persistence) GetMapBlocks(
	minLatitude,
	minLongitude,
	maxLatitude,
	maxLongitude decimal.Decimal,
) ([]MapBlock, error) {
	rows, err := svc.db.Query(
		getMapBlocksQuery,
		minLatitude.Sub(coordinateOvershoot),
		maxLatitude.Add(coordinateOvershoot),
		minLongitude.Sub(coordinateOvershoot),
		maxLongitude.Add(coordinateOvershoot))
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
	latitude = $1
	AND longitude = $2
`

func (svc Persistence) GetMapBlock(latitude, longitude decimal.Decimal) (*MapBlock, error) {
	var block MapBlock
	err := svc.db.QueryRow(
		getMapBlockQuery,
		segmentCoordinate(latitude),
		segmentCoordinate(longitude),
	).Scan(
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
VALUES ($1, $2)
ON CONFLICT DO NOTHING
`

func (svc Persistence) InsertMapBlock(latitude, longitude decimal.Decimal) error {
	_, err := svc.db.Exec(
		insertMapBlockQuery,
		segmentCoordinate(latitude),
		segmentCoordinate(longitude))
	return err
}

const insertImageQuery = `
INSERT INTO images (public_id, format)
VALUES ($1, $2)
ON CONFLICT DO NOTHING
`

func (svc Persistence) InsertImage(publicID, format string) error {
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

func (svc Persistence) UpdateImage(publicID, status string) error {
	_, err := svc.db.Exec(updateImageQuery, publicID, status)
	return err
}

type Car struct {
	Year  int
	Make  string
	Model string
	Trim  string
	Color string
	Image CloudinaryImage
}

// TODO: Fetch additional columns, order by created descending,
// and paginate.
// TODO: Rename "brand" to "make".
const getCarsQuery = `
SELECT
	c.year,
	c.brand,
	c.model,
	c.trim,
	c.color,
	i.public_id,
	i.format
FROM cars c
LEFT JOIN images i ON
	i.public_id = c.images_public_id
	AND i.status = 'approved'
WHERE c.map_block_id = $1
ORDER BY c.created DESC
`

func (svc Persistence) GetCars(mapBlockID int) ([]Car, error) {
	rows, err := svc.db.Query(getCarsQuery, mapBlockID)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to read cars")
	}
	cars := make([]Car, 0, 10)
	for rows.Next() {
		var car Car
		var publicID sql.NullString
		var format sql.NullString
		err := rows.Scan(
			&car.Year,
			&car.Make,
			&car.Model,
			&car.Trim,
			&car.Color,
			&publicID,
			&format)
		if err != nil {
			return nil, errors.WithMessage(err, "failed to scan car row into struct")
		}
		// TODO: If the image is awaiting moderation, add an "awaiting moderation" image.
		// TODO: If there is no image, add a stock photo.
		if publicID.Valid && format.Valid {
			car.Image.PublicID = publicID.String
			car.Image.Format = format.String
		}
		cars = append(cars, car)
	}
	return cars, nil
}

var nonAlphanumeric = regexp.MustCompile(`[^\w\d]`)

func (svc Persistence) licenseHash(licenseState, licensePlate string) (string, error) {
	license := []byte(fmt.Sprintf("%s-%s",
		strings.ToUpper(strings.TrimSpace(licenseState)),
		strings.ToUpper(nonAlphanumeric.ReplaceAllString(licensePlate, ""))))
	hash, err := scrypt.Key(license, svc.salt, 1<<15, 8, 1, 32)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(hash), nil
}

const insertCarQuery = `
INSERT INTO cars(
	license_hash,
	map_block_id,
	year,
	brand,
	model,
	trim,
	color,
	images_public_id
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
`

func (svc Persistence) InsertCar(
	licenseState,
	licensePlate string,
	mapBlockID,
	year int,
	make,
	model,
	trim,
	color,
	imagePublicID string,
) error {
	hash, err := svc.licenseHash(licenseState, licensePlate)
	if err != nil {
		return errors.WithMessage(err, "error generating license hash")
	}
	_, err = svc.db.Exec(
		insertCarQuery,
		hash,
		mapBlockID,
		year,
		strings.TrimSpace(make),
		strings.TrimSpace(model),
		strings.TrimSpace(trim),
		strings.ToLower(strings.TrimSpace(color)),
		strings.TrimSpace(imagePublicID))
	if err != nil {
		return errors.WithMessage(err, "failed to insert car")
	}
	return nil
}
