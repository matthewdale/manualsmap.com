package images

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSignature(t *testing.T) {
	signer := NewService("abcd")
	sig := signer.Signature(map[string]string{
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
