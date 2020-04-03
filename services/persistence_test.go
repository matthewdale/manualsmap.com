package services

import (
	"fmt"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestSegmentCoordinate(t *testing.T) {
	tests := []struct {
		description string
		coordinate  decimal.Decimal
		expected    decimal.Decimal
	}{
		{
			description: "Coordinates should be segmented to 0.05 degrees",
			coordinate:  decimal.NewFromFloat(123.45678),
			expected:    decimal.NewFromFloat(123.45),
		},
		{
			description: "Coordinates already segmented to 0.05 should stay the same",
			coordinate:  decimal.NewFromFloat(123.45),
			expected:    decimal.NewFromFloat(123.45),
		},
		{
			description: "Coordinates slightly below a segment should be truncated",
			coordinate:  decimal.NewFromFloat(123.449999),
			expected:    decimal.NewFromFloat(123.4),
		},
		{
			description: "Negative coordinates slightly below a segment should be truncated",
			coordinate:  decimal.NewFromFloat(-123.449999),
			expected:    decimal.NewFromFloat(-123.4),
		},
	}

	for _, test := range tests {
		test := test // Capture range variable.
		t.Run(test.description, func(t *testing.T) {
			actual := segmentCoordinate(test.coordinate)
			assert.True(
				t,
				test.expected.Equals(actual),
				fmt.Sprintf(
					"Expected segmented coordinate to match (want %s, got %s)",
					test.expected.String(),
					actual.String()))
		})
	}
}
