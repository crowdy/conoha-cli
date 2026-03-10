package model

import (
	"encoding/json"
	"testing"
	"time"
)

func TestFlexTimeUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    time.Time
		wantErr bool
	}{
		{
			name:  "RFC3339",
			input: `"2025-10-18T01:52:32Z"`,
			want:  time.Date(2025, 10, 18, 1, 52, 32, 0, time.UTC),
		},
		{
			name:  "RFC3339 with offset",
			input: `"2025-10-18T01:52:32+09:00"`,
			want:  time.Date(2025, 10, 18, 1, 52, 32, 0, time.FixedZone("", 9*60*60)),
		},
		{
			name:  "without timezone microseconds",
			input: `"2025-10-18T01:52:32.000000"`,
			want:  time.Date(2025, 10, 18, 1, 52, 32, 0, time.UTC),
		},
		{
			name:  "without timezone no fractional",
			input: `"2025-10-18T01:52:32"`,
			want:  time.Date(2025, 10, 18, 1, 52, 32, 0, time.UTC),
		},
		{
			name:  "null",
			input: `"null"`,
			want:  time.Time{},
		},
		{
			name:  "empty string",
			input: `""`,
			want:  time.Time{},
		},
		{
			name:    "invalid format",
			input:   `"not-a-date"`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ft FlexTime
			err := json.Unmarshal([]byte(tt.input), &ft)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !ft.Equal(tt.want) {
				t.Errorf("got %v, want %v", ft, tt.want)
			}
		})
	}
}

func TestFlexTimeMarshalJSON(t *testing.T) {
	ft := FlexTime{Time: time.Date(2025, 10, 18, 1, 52, 32, 0, time.UTC)}
	b, err := json.Marshal(ft)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := `"2025-10-18T01:52:32Z"`
	if string(b) != want {
		t.Errorf("got %s, want %s", string(b), want)
	}
}

func TestFlexTimeMarshalJSONZero(t *testing.T) {
	ft := FlexTime{}
	b, err := json.Marshal(ft)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(b) != `""` {
		t.Errorf("got %s, want empty string", string(b))
	}
}

func TestFlexTimeInStruct(t *testing.T) {
	// Simulate ConoHa Block Storage API response
	jsonData := `{"id":"vol-1","created_at":"2025-10-18T01:52:32.000000"}`
	var v struct {
		ID        string   `json:"id"`
		CreatedAt FlexTime `json:"created_at"`
	}
	if err := json.Unmarshal([]byte(jsonData), &v); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v.ID != "vol-1" {
		t.Errorf("got ID %q, want %q", v.ID, "vol-1")
	}
	want := time.Date(2025, 10, 18, 1, 52, 32, 0, time.UTC)
	if !v.CreatedAt.Equal(want) {
		t.Errorf("got %v, want %v", v.CreatedAt.Time, want)
	}
}
