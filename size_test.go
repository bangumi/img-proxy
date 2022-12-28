package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseSize(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		input string
		err   bool
		size  Size
	}{
		{
			input: "200",
			size:  Size{Width: 200},
		},
		{
			input: "2xx00",
			err:   true,
		},
		{
			input: "2xx300",
			err:   true,
		},
		{
			input: "2x300",
			size:  Size{Width: 2, Height: 300},
		},
	}

	for _, tc := range testCases {
		actual, err := ParseSize(tc.input)
		if tc.err {
			require.NotNil(t, err)
		} else {
			require.Equal(t, tc.size, actual)
		}
	}
}
