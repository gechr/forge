package forge

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExpandPRList(t *testing.T) {
	t.Parallel()

	tests := []struct {
		spec string
		want []int
	}{
		{spec: "1", want: []int{1}},
		{spec: "1,2,5-7", want: []int{1, 2, 5, 6, 7}},
		{spec: "3-3", want: []int{3}},
		{spec: "2,1", want: []int{2, 1}},
		{spec: "1,1,1-2", want: []int{1, 2}},
		{spec: " 1 , 2 ", want: []int{1, 2}},
	}

	for _, test := range tests {
		t.Run(test.spec, func(t *testing.T) {
			t.Parallel()

			got, err := ExpandPRList(test.spec)
			require.NoError(t, err)
			require.Equal(t, test.want, got)
		})
	}
}

func TestExpandPRListRejectsInvalid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		spec    string
		wantErr string
	}{
		{name: "empty", spec: "", wantErr: `empty PR number in ""`},
		{name: "empty segment", spec: "1,,2", wantErr: `empty PR number in "1,,2"`},
		{name: "zero", spec: "0", wantErr: `invalid PR reference "0"`},
		{name: "negative", spec: "-1", wantErr: `invalid PR reference "-1"`},
		{name: "reversed range", spec: "5-3", wantErr: `invalid PR reference "5-3"`},
		{name: "not a number", spec: "abc", wantErr: `invalid PR reference "abc"`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			_, err := ExpandPRList(test.spec)
			require.EqualError(t, err, test.wantErr)
		})
	}
}
