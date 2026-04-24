package output

import (
	"reflect"
	"testing"
)

type filterTestItem struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Count  int    `json:"count"`
}

func TestFilterRows(t *testing.T) {
	data := []filterTestItem{
		{Name: "a", Status: "ACTIVE", Count: 1},
		{Name: "b", Status: "STOPPED", Count: 2},
		{Name: "c", Status: "ACTIVE", Count: 3},
	}

	t.Run("single filter", func(t *testing.T) {
		result, err := FilterRows(data, []string{"status=ACTIVE"})
		if err != nil {
			t.Fatal(err)
		}
		rows := result.([]filterTestItem)
		if len(rows) != 2 {
			t.Fatalf("expected 2 rows, got %d", len(rows))
		}
		if rows[0].Name != "a" || rows[1].Name != "c" {
			t.Errorf("unexpected rows: %+v", rows)
		}
	})

	t.Run("multiple filters (AND)", func(t *testing.T) {
		result, err := FilterRows(data, []string{"status=ACTIVE", "name=c"})
		if err != nil {
			t.Fatal(err)
		}
		rows := result.([]filterTestItem)
		if len(rows) != 1 || rows[0].Name != "c" {
			t.Errorf("expected [c], got %+v", rows)
		}
	})

	t.Run("case insensitive", func(t *testing.T) {
		result, err := FilterRows(data, []string{"status=active"})
		if err != nil {
			t.Fatal(err)
		}
		rows := result.([]filterTestItem)
		if len(rows) != 2 {
			t.Fatalf("expected 2 rows, got %d", len(rows))
		}
	})

	t.Run("no match", func(t *testing.T) {
		result, err := FilterRows(data, []string{"name=z"})
		if err != nil {
			t.Fatal(err)
		}
		rows := result.([]filterTestItem)
		if len(rows) != 0 {
			t.Errorf("expected 0 rows, got %d", len(rows))
		}
	})

	t.Run("unknown field", func(t *testing.T) {
		_, err := FilterRows(data, []string{"unknown=x"})
		if err == nil {
			t.Error("expected error for unknown field")
		}
	})

	t.Run("invalid format", func(t *testing.T) {
		_, err := FilterRows(data, []string{"badfilter"})
		if err == nil {
			t.Error("expected error for invalid filter format")
		}
	})

	t.Run("contains operator", func(t *testing.T) {
		bigger := []filterTestItem{
			{Name: "ubuntu-24.04", Status: "ACTIVE", Count: 1},
			{Name: "debian-12", Status: "ACTIVE", Count: 2},
			{Name: "ubuntu-22.04", Status: "STOPPED", Count: 3},
		}
		result, err := FilterRows(bigger, []string{"name~ubuntu"})
		if err != nil {
			t.Fatal(err)
		}
		rows := result.([]filterTestItem)
		if len(rows) != 2 {
			t.Fatalf("expected 2 rows, got %d", len(rows))
		}
		if rows[0].Name != "ubuntu-24.04" || rows[1].Name != "ubuntu-22.04" {
			t.Errorf("unexpected rows: %+v", rows)
		}
	})

	t.Run("contains is case insensitive", func(t *testing.T) {
		result, err := FilterRows(data, []string{"status~ACTI"})
		if err != nil {
			t.Fatal(err)
		}
		rows := result.([]filterTestItem)
		if len(rows) != 2 {
			t.Fatalf("expected 2 rows, got %d", len(rows))
		}
	})

	t.Run("regex operator", func(t *testing.T) {
		bigger := []filterTestItem{
			{Name: "ubuntu-24.04", Status: "ACTIVE", Count: 1},
			{Name: "ubuntu-22.04", Status: "STOPPED", Count: 2},
			{Name: "debian-12", Status: "ACTIVE", Count: 3},
		}
		result, err := FilterRows(bigger, []string{`name~=^ubuntu-\d+\.\d+$`})
		if err != nil {
			t.Fatal(err)
		}
		rows := result.([]filterTestItem)
		if len(rows) != 2 {
			t.Fatalf("expected 2 rows, got %d", len(rows))
		}
	})

	t.Run("regex invalid", func(t *testing.T) {
		_, err := FilterRows(data, []string{"name~=[unclosed"})
		if err == nil {
			t.Error("expected error for invalid regex")
		}
	})

	t.Run("empty key", func(t *testing.T) {
		for _, f := range []string{"=value", "~value", "~=value"} {
			if _, err := FilterRows(data, []string{f}); err == nil {
				t.Errorf("expected error for empty key filter %q", f)
			}
		}
	})

	t.Run("combined operators AND", func(t *testing.T) {
		bigger := []filterTestItem{
			{Name: "ubuntu-24.04", Status: "ACTIVE", Count: 1},
			{Name: "ubuntu-22.04", Status: "STOPPED", Count: 2},
			{Name: "debian-12", Status: "ACTIVE", Count: 3},
		}
		result, err := FilterRows(bigger, []string{"name~ubuntu", "status=ACTIVE"})
		if err != nil {
			t.Fatal(err)
		}
		rows := result.([]filterTestItem)
		if len(rows) != 1 || rows[0].Name != "ubuntu-24.04" {
			t.Errorf("expected [ubuntu-24.04 ACTIVE], got %+v", rows)
		}
	})

	t.Run("empty filters", func(t *testing.T) {
		result, err := FilterRows(data, nil)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(result, data) {
			t.Error("expected original data returned")
		}
	})

	t.Run("non-slice", func(t *testing.T) {
		item := filterTestItem{Name: "x"}
		result, err := FilterRows(item, []string{"name=x"})
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(result, item) {
			t.Error("expected original item returned for non-slice")
		}
	})
}

func TestSortRows(t *testing.T) {
	data := []filterTestItem{
		{Name: "charlie", Status: "ACTIVE", Count: 3},
		{Name: "alpha", Status: "STOPPED", Count: 1},
		{Name: "bravo", Status: "ACTIVE", Count: 2},
	}

	t.Run("sort by string", func(t *testing.T) {
		result, err := SortRows(data, "name")
		if err != nil {
			t.Fatal(err)
		}
		rows := result.([]filterTestItem)
		if rows[0].Name != "alpha" || rows[1].Name != "bravo" || rows[2].Name != "charlie" {
			t.Errorf("unexpected sort order: %+v", rows)
		}
	})

	t.Run("sort by int", func(t *testing.T) {
		result, err := SortRows(data, "count")
		if err != nil {
			t.Fatal(err)
		}
		rows := result.([]filterTestItem)
		if rows[0].Count != 1 || rows[1].Count != 2 || rows[2].Count != 3 {
			t.Errorf("unexpected sort order: %+v", rows)
		}
	})

	t.Run("unknown field", func(t *testing.T) {
		_, err := SortRows(data, "unknown")
		if err == nil {
			t.Error("expected error for unknown field")
		}
	})

	t.Run("empty sort-by", func(t *testing.T) {
		result, err := SortRows(data, "")
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(result, data) {
			t.Error("expected original data returned")
		}
	})

	t.Run("non-slice", func(t *testing.T) {
		item := filterTestItem{Name: "x"}
		result, err := SortRows(item, "name")
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(result, item) {
			t.Error("expected original item returned for non-slice")
		}
	})

	t.Run("does not mutate original", func(t *testing.T) {
		original := make([]filterTestItem, len(data))
		copy(original, data)
		_, err := SortRows(data, "name")
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(data, original) {
			t.Error("original data was mutated")
		}
	})
}
