package images

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUploadSignature(t *testing.T) {
	svc := NewService(nil, "abcd")
	sig := svc.UploadSignature(map[string]string{
		"timestamp": "1315060510",
		"public_id": "sample_image",
		"eager":     "w_400,h_300,c_pad|w_260,h_200,c_crop",
	})
	assert.Equal(
		t,
		"bfd09f95f331f558cbd1320e67aa8d488770583e",
		sig,
		"Expected signatures to match")
}
func TestNotificationSignature(t *testing.T) {
	svc := NewService(nil, "abcd")
	body := `{"public_id":"djhoeaqcynvogt9xzbn9","version":1368881626,"width":864,"height":576,"format":"jpg","resource_type":"image","created_at":"2013-05-18T12:53:46Z","bytes":120253,"type":"upload","url":"https://res.cloudinary.com/1233456ab/image/upload/v1368881626/djhoeaqcynvogt9xzbn9.jpg","secure_url":"https://cloudinary-a.akamaihd.net/1233456ab/image/upload/v1368881626/djhoeaqcynvogt9xzbn9.jpg"}`
	timestamp := "1368881627"
	sig := svc.NotificationSignature(body, timestamp)
	assert.Equal(
		t,
		"0f2cb563a8edbdd6bc865a8c3d14fe9bdbebc1a3",
		sig,
		"Expected signatures to match")
}
