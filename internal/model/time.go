package model

import (
	"fmt"
	"strings"
	"time"
)

// FlexTime is a time.Time wrapper that handles multiple timestamp formats.
// ConoHa APIs return timestamps in varying formats (with/without timezone).
type FlexTime struct{ time.Time }

func (t *FlexTime) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), `"`)
	if s == "null" || s == "" {
		return nil
	}
	for _, layout := range []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02T15:04:05.000000",
		"2006-01-02T15:04:05",
	} {
		if parsed, err := time.Parse(layout, s); err == nil {
			t.Time = parsed
			return nil
		}
	}
	return fmt.Errorf("cannot parse time %q", s)
}

func (t FlexTime) MarshalJSON() ([]byte, error) {
	if t.IsZero() {
		return []byte(`""`), nil
	}
	return []byte(`"` + t.Time.Format(time.RFC3339) + `"`), nil
}
