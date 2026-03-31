package server

import (
	"testing"
)

func TestSecurityGroupCommandsRegistered(t *testing.T) {
	cmds := Cmd.Commands()
	found := map[string]bool{}
	for _, c := range cmds {
		found[c.Name()] = true
	}
	for _, name := range []string{"add-security-group", "remove-security-group"} {
		if !found[name] {
			t.Errorf("command %q not registered under server", name)
		}
	}
}

func TestAddSecurityGroupNameFlagRequired(t *testing.T) {
	if !addSecurityGroupCmd.HasFlags() {
		t.Fatal("add-security-group has no flags")
	}
	f := addSecurityGroupCmd.Flags().Lookup("name")
	if f == nil {
		t.Fatal("add-security-group missing --name flag")
	}
	ann := f.Annotations["cobra_annotation_bash_completion_one_required_flag"]
	if len(ann) == 0 {
		t.Error("--name flag on add-security-group is not marked as required")
	}
}

func TestRemoveSecurityGroupNameFlagRequired(t *testing.T) {
	if !removeSecurityGroupCmd.HasFlags() {
		t.Fatal("remove-security-group has no flags")
	}
	f := removeSecurityGroupCmd.Flags().Lookup("name")
	if f == nil {
		t.Fatal("remove-security-group missing --name flag")
	}
	ann := f.Annotations["cobra_annotation_bash_completion_one_required_flag"]
	if len(ann) == 0 {
		t.Error("--name flag on remove-security-group is not marked as required")
	}
}
