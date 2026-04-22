package server

import (
	"reflect"
	"testing"
)

func TestParsePortRanges(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    []portRange
		wantErr bool
	}{
		{"single", "7860", []portRange{{7860, 7860}}, false},
		{"comma list", "7860,8080", []portRange{{7860, 7860}, {8080, 8080}}, false},
		{"range", "9000-9010", []portRange{{9000, 9010}}, false},
		{"mixed", "7860,8080,9000-9010", []portRange{{7860, 7860}, {8080, 8080}, {9000, 9010}}, false},
		{"whitespace tolerant", " 7860 , 8080 ", []portRange{{7860, 7860}, {8080, 8080}}, false},
		{"trailing comma", "7860,", []portRange{{7860, 7860}}, false},

		{"empty", "", nil, true},
		{"empty only commas", ",,,", nil, true},
		{"non-numeric", "abc", nil, true},
		{"out of range low", "0", nil, true},
		{"out of range high", "65536", nil, true},
		{"bad range", "8080-8000", nil, true},
		{"range with non-number", "8080-abc", nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parsePortRanges(tt.in)
			if (err != nil) != tt.wantErr {
				t.Errorf("err=%v wantErr=%v", err, tt.wantErr)
			}
			if !tt.wantErr && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}
