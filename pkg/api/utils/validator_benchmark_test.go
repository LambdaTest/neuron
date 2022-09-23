package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func BenchmarkValidateInterval(b *testing.B) {
	type args struct {
		intervalType string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{name: "valid_input", args: args{intervalType: "Week"}, want: "WEEK", wantErr: false},
		{name: "invalid_input", args: args{intervalType: "Year"}, want: "", wantErr: true},
	}
	for i := 0; i < b.N; i++ {
		for _, tt := range tests {
			res, err := validateInterval(tt.args.intervalType)
			assert.Equal(b, res, tt.want)
			assert.Equal(b, err != nil, tt.wantErr)
		}
	}
}
