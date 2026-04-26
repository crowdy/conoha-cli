package server

import (
	"fmt"
	"sort"
	"strings"

	"github.com/crowdy/conoha-cli/internal/api"
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

// resolvePresetImage queries ListImages and returns the lexicographically
// newest active image whose name satisfies match. ConoHa rotates these
// images periodically, so resolution at preset-apply time avoids stale
// hardcoded IDs in the CLI binary.
func resolvePresetImage(imageAPI *api.ImageAPI, match func(string) bool) (string, error) {
	images, err := imageAPI.ListImages()
	if err != nil {
		return "", fmt.Errorf("listing images: %w", err)
	}
	var matched []string // names
	idByName := make(map[string]string)
	for _, img := range images {
		if img.Status != "active" {
			continue
		}
		if !match(img.Name) {
			continue
		}
		matched = append(matched, img.Name)
		idByName[img.Name] = img.ID
	}
	if len(matched) == 0 {
		return "", fmt.Errorf("no image matched preset criteria (try `conoha image list` to see what is available)")
	}
	sort.Sort(sort.Reverse(sort.StringSlice(matched)))
	return idByName[matched[0]], nil
}

// validatePresetSecurityGroups returns nil if every name in want exists in
// the tenant's security-group list. On a missing entry it returns an error
// listing the missing names plus the actual SG list, so the operator can
// self-diagnose without rerunning `conoha server list-sg`.
func validatePresetSecurityGroups(networkAPI *api.NetworkAPI, want []string) error {
	sgs, err := networkAPI.ListSecurityGroups()
	if err != nil {
		return fmt.Errorf("listing security groups: %w", err)
	}
	have := make(map[string]bool, len(sgs))
	names := make([]string, 0, len(sgs))
	for _, sg := range sgs {
		if sg.Name == "" {
			continue
		}
		have[sg.Name] = true
		names = append(names, sg.Name)
	}
	var missing []string
	for _, w := range want {
		if !have[w] {
			missing = append(missing, w)
		}
	}
	if len(missing) == 0 {
		return nil
	}
	sort.Strings(names)
	return fmt.Errorf("preset security groups not found: %s (available: %s)",
		strings.Join(missing, ", "), strings.Join(names, ", "))
}
