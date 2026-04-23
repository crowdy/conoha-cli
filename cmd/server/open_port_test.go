package server

import (
	"reflect"
	"testing"

	"github.com/crowdy/conoha-cli/internal/model"
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
		{"dedup singles", "7860,7860", []portRange{{7860, 7860}}, false},
		{"dedup range and single", "80,80-80", []portRange{{80, 80}}, false},
		{"dedup ranges", "9000-9010,9000-9010", []portRange{{9000, 9010}}, false},
		{"dedup preserves order", "8080,7860,8080", []portRange{{8080, 8080}, {7860, 7860}}, false},

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

func TestEthertypeFromCIDR(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    string
		wantErr bool
	}{
		{"ipv4 any", "0.0.0.0/0", "IPv4", false},
		{"ipv4 /32", "10.0.0.1/32", "IPv4", false},
		{"ipv4 /8", "10.0.0.0/8", "IPv4", false},
		{"ipv6 any", "::/0", "IPv6", false},
		{"ipv6 /128", "2001:db8::1/128", "IPv6", false},
		{"ipv6 /64", "2001:db8::/64", "IPv6", false},

		{"empty", "", "", true},
		{"bare ip no mask", "10.0.0.1", "", true},
		{"bad cidr", "10.0.0/8", "", true},
		{"garbage", "not-an-ip", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ethertypeFromCIDR(tt.in)
			if (err != nil) != tt.wantErr {
				t.Errorf("err=%v wantErr=%v", err, tt.wantErr)
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFilterExistingRanges(t *testing.T) {
	ptr := func(n int) *int { return &n }
	existing := []model.SecurityGroupRule{
		{Direction: "ingress", Protocol: "tcp", EtherType: "IPv4", RemoteIPPrefix: "0.0.0.0/0", PortRangeMin: ptr(80), PortRangeMax: ptr(80)},
		{Direction: "ingress", Protocol: "tcp", EtherType: "IPv4", RemoteIPPrefix: "0.0.0.0/0", PortRangeMin: ptr(9000), PortRangeMax: ptr(9010)},
		// should not match (different CIDR)
		{Direction: "ingress", Protocol: "tcp", EtherType: "IPv4", RemoteIPPrefix: "10.0.0.0/8", PortRangeMin: ptr(443), PortRangeMax: ptr(443)},
		// should not match (udp)
		{Direction: "ingress", Protocol: "udp", EtherType: "IPv4", RemoteIPPrefix: "0.0.0.0/0", PortRangeMin: ptr(53), PortRangeMax: ptr(53)},
		// should not match (egress)
		{Direction: "egress", Protocol: "tcp", EtherType: "IPv4", RemoteIPPrefix: "0.0.0.0/0", PortRangeMin: ptr(22), PortRangeMax: ptr(22)},
		// nil ports should not crash the filter
		{Direction: "ingress", Protocol: "tcp", EtherType: "IPv4", RemoteIPPrefix: "0.0.0.0/0", PortRangeMin: nil, PortRangeMax: nil},
	}

	input := []portRange{{80, 80}, {443, 443}, {9000, 9010}, {22, 22}}
	wantNew := []portRange{{443, 443}, {22, 22}}
	wantSkipped := []portRange{{80, 80}, {9000, 9010}}

	gotNew, gotSkipped := filterExistingRanges(input, existing, "tcp", "0.0.0.0/0", "IPv4")
	if !reflect.DeepEqual(gotNew, wantNew) {
		t.Errorf("new: got %v, want %v", gotNew, wantNew)
	}
	if !reflect.DeepEqual(gotSkipped, wantSkipped) {
		t.Errorf("skipped: got %v, want %v", gotSkipped, wantSkipped)
	}
}

func TestPickSGByName(t *testing.T) {
	sgs := []model.SecurityGroup{
		{ID: "id-a", Name: "foo-sg"},
		{ID: "id-b", Name: "bar-sg"},
		{ID: "id-c", Name: "foo-sg"},
	}

	t.Run("single match", func(t *testing.T) {
		got, dupes := pickSGByName(sgs, "bar-sg")
		if got == nil || got.ID != "id-b" {
			t.Errorf("got %+v", got)
		}
		if len(dupes) != 0 {
			t.Errorf("dupes %v, want none", dupes)
		}
	})

	t.Run("multiple matches returns first and lists duplicate IDs", func(t *testing.T) {
		got, dupes := pickSGByName(sgs, "foo-sg")
		if got == nil || got.ID != "id-a" {
			t.Errorf("got %+v", got)
		}
		if !reflect.DeepEqual(dupes, []string{"id-c"}) {
			t.Errorf("dupes %v", dupes)
		}
	})

	t.Run("no match", func(t *testing.T) {
		got, dupes := pickSGByName(sgs, "nope")
		if got != nil {
			t.Errorf("got %+v, want nil", got)
		}
		if len(dupes) != 0 {
			t.Errorf("dupes %v", dupes)
		}
	})
}
