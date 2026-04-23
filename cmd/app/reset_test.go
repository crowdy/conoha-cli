package app

import (
	"strings"
	"testing"
)

func TestResetConfirmationMessage(t *testing.T) {
	cases := []struct {
		name        string
		mode        Mode
		legacy      bool
		mustContain []string
		mustNot     []string
	}{
		{
			name:   "proxy mode shows all proxy-specific side effects",
			mode:   ModeProxy,
			legacy: false,
			mustContain: []string{
				`"myapp"`,
				"server-01",
				"mode=proxy",
				"compose down",
				"/opt/conoha/myapp",
				".env.server",
				"proxy registration will be dropped",
				"rollback window will be discarded",
			},
		},
		{
			name:   "no-proxy mode omits proxy-specific lines",
			mode:   ModeNoProxy,
			legacy: false,
			mustContain: []string{
				"mode=no-proxy",
				"compose down",
			},
			mustNot: []string{
				"proxy registration",
				"rollback window",
			},
		},
		{
			name:   "legacy fallback surfaces the 'legacy, no marker' hint",
			mode:   ModeProxy,
			legacy: true,
			mustContain: []string{
				"mode=proxy (legacy, no marker)",
				"proxy registration will be dropped",
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			msg := resetConfirmationMessage("myapp", "server-01", tc.mode, tc.legacy)
			for _, want := range tc.mustContain {
				if !strings.Contains(msg, want) {
					t.Errorf("missing %q in:\n%s", want, msg)
				}
			}
			for _, forbidden := range tc.mustNot {
				if strings.Contains(msg, forbidden) {
					t.Errorf("should not contain %q in no-proxy mode:\n%s", forbidden, msg)
				}
			}
		})
	}
}
