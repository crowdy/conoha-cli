package server

import (
	"sort"
	"strings"
)

// presetSpec describes the values a `--for <name>` preset fills in
// when the corresponding explicit flag is empty.
type presetSpec struct {
	Flavor         string
	SecurityGroups []string
	ImageMatch     func(name string) bool
}

var presets = map[string]presetSpec{
	"proxy": {
		Flavor:         "g2l-t-c3m2",
		SecurityGroups: []string{"default", "IPv4v6-SSH", "IPv4v6-Web", "IPv4v6-ICMP"},
		ImageMatch:     matchDockerUbuntuAmd64,
	},
}

// matchDockerUbuntuAmd64 returns true for ConoHa images named like
// "vmi-docker-<version>-ubuntu-<release>-amd64".
func matchDockerUbuntuAmd64(name string) bool {
	return strings.HasPrefix(name, "vmi-docker-") &&
		strings.Contains(name, "-ubuntu-") &&
		strings.HasSuffix(name, "-amd64")
}

// knownPresetList returns a sorted, comma-joined list of preset names,
// suitable for inclusion in error messages.
func knownPresetList() string {
	names := make([]string, 0, len(presets))
	for n := range presets {
		names = append(names, n)
	}
	sort.Strings(names)
	return strings.Join(names, ", ")
}
