package services

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUploadSignature(t *testing.T) {
	svc := NewCloudinary("abcd")
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
	svc := NewCloudinary("abcd")
	body := `{"public_id":"djhoeaqcynvogt9xzbn9","version":1368881626,"width":864,"height":576,"format":"jpg","resource_type":"image","created_at":"2013-05-18T12:53:46Z","bytes":120253,"type":"upload","url":"https://res.cloudinary.com/1233456ab/image/upload/v1368881626/djhoeaqcynvogt9xzbn9.jpg","secure_url":"https://cloudinary-a.akamaihd.net/1233456ab/image/upload/v1368881626/djhoeaqcynvogt9xzbn9.jpg"}`
	timestamp := "1368881627"
	sig := svc.NotificationSignature(body, timestamp)
	assert.Equal(
		t,
		"0f2cb563a8edbdd6bc865a8c3d14fe9bdbebc1a3",
		sig,
		"Expected signatures to match")
}

func TestDeliverySignature(t *testing.T) {
	svc := NewCloudinary("abcd")
	img := CloudinaryImage{
		PublicID: "sample",
		Format:   "png",
	}
	transform := "w_300,h_250,e_grayscale"
	sig := svc.deliverySignature(img, transform)
	assert.Equal(
		t,
		"s--INQUGulu--",
		sig,
		"Expected signatures to match")
}

func TestURL(t *testing.T) {
	tests := []struct {
		description string
		img         CloudinaryImage
		transform   string
		expected    *url.URL
	}{
		{
			description: "Cloudinary documentation example",
			img: CloudinaryImage{
				PublicID: "sample",
				Format:   "png",
			},
			transform: "w_300,h_250,e_grayscale",
			expected: &url.URL{
				Scheme: "https",
				Host:   "res.cloudinary.com",
				Path:   "dawfgqsur/image/authenticated/s--INQUGulu--/w_300,h_250,e_grayscale/sample.png",
			},
		},
		{
			description: "Should work with no transform",
			img: CloudinaryImage{
				PublicID: "sample",
				Format:   "png",
			},
			transform: "",
			expected: &url.URL{
				Scheme: "https",
				Host:   "res.cloudinary.com",
				Path:   "dawfgqsur/image/authenticated/s--8u3FOpeL--/sample.png",
			},
		},
	}
	svc := NewCloudinary("abcd")

	for _, test := range tests {
		test := test // Capture range variable.
		t.Run(test.description, func(t *testing.T) {
			t.Parallel()

			actual := svc.URL(test.img, test.transform)
			assert.Equal(
				t,
				test.expected,
				actual,
				"Expected URLs to match")
		})
	}
}
