package commands

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHeightRangeDivide(t *testing.T) {
	testCases := []struct {
		hr   heightRange
		n    int
		want []heightRange
	}{
		{
			hr: heightRange{min: 1, max: 10},
			n:  2,
			want: []heightRange{
				{min: 1, max: 5},
				{min: 6, max: 10},
			},
		},
		{
			hr: heightRange{min: 1, max: 3},
			n:  3,
			want: []heightRange{
				{min: 1, max: 1},
				{min: 2, max: 2},
				{min: 3, max: 3},
			},
		},
		{
			hr: heightRange{min: 1, max: 4},
			n:  3,
			want: []heightRange{
				{min: 1, max: 1},
				{min: 2, max: 2},
				{min: 3, max: 4},
			},
		},
		{
			hr: heightRange{min: 1, max: 4},
			n:  2,
			want: []heightRange{
				{min: 1, max: 2},
				{min: 3, max: 4},
			},
		},
		{
			hr: heightRange{min: 1, max: 4},
			n:  1,
			want: []heightRange{
				{min: 1, max: 4},
			},
		},
		{
			hr: heightRange{min: 0, max: math.MaxInt64},
			n:  1,
			want: []heightRange{
				{min: 0, max: math.MaxInt64},
			},
		},
		{
			hr: heightRange{min: 0, max: math.MaxInt64},
			n:  2,
			want: []heightRange{
				{min: 0, max: 4611686018427387902},
				{min: 4611686018427387903, max: math.MaxInt64},
			},
		},
	}

	for _, tc := range testCases {
		t.Run("", func(t *testing.T) {
			got := tc.hr.divide(tc.n)
			assert.Equal(t, tc.want, got)
		})
	}
}
