package utils

import "testing"

func TestValidateInterval(t *testing.T) {
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
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := validateInterval(tt.args.intervalType)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateInterval() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ValidateInterval() got = %v, want %v", got, tt.want)
			}
		})
	}
}
