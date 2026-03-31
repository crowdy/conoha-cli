# Server Security Group Commands Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `server add-security-group` and `server remove-security-group` commands to manage security group assignments on existing servers.

**Architecture:** Two new API wrapper methods on ComputeAPI that delegate to the existing `ServerAction()` method. One new command file `cmd/server/security_group.go` with two cobra commands registered in `server.go`. Both commands require `prompt.Confirm()`.

**Tech Stack:** Go, cobra, httptest

---

### Task 1: Add API methods to ComputeAPI

**Files:**
- Modify: `internal/api/compute.go:134` (after RebuildServer)
- Test: `internal/api/compute_test.go`

- [ ] **Step 1: Write failing tests for AddSecurityGroup and RemoveSecurityGroup**

Add to `internal/api/compute_test.go`:

```go
func TestAddSecurityGroup(t *testing.T) {
	const serverID = "sg-add-srv-id"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/v2.1/servers/"+serverID+"/action") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		sg, ok := body["addSecurityGroup"].(map[string]any)
		if !ok {
			t.Errorf("expected 'addSecurityGroup' key in body, got %v", body)
		} else if sg["name"] != "my-sg" {
			t.Errorf("expected name 'my-sg', got %v", sg["name"])
		}
		w.WriteHeader(http.StatusAccepted)
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	api := NewComputeAPI(newTestClient(ts))
	if err := api.AddSecurityGroup(serverID, "my-sg"); err != nil {
		t.Fatalf("AddSecurityGroup() error: %v", err)
	}
}

func TestRemoveSecurityGroup(t *testing.T) {
	const serverID = "sg-rm-srv-id"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/v2.1/servers/"+serverID+"/action") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		sg, ok := body["removeSecurityGroup"].(map[string]any)
		if !ok {
			t.Errorf("expected 'removeSecurityGroup' key in body, got %v", body)
		} else if sg["name"] != "old-sg" {
			t.Errorf("expected name 'old-sg', got %v", sg["name"])
		}
		w.WriteHeader(http.StatusAccepted)
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	api := NewComputeAPI(newTestClient(ts))
	if err := api.RemoveSecurityGroup(serverID, "old-sg"); err != nil {
		t.Fatalf("RemoveSecurityGroup() error: %v", err)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/api/ -run 'TestAddSecurityGroup|TestRemoveSecurityGroup' -v`
Expected: compilation error — `AddSecurityGroup` and `RemoveSecurityGroup` not defined

- [ ] **Step 3: Implement API methods**

Add to `internal/api/compute.go` after `RebuildServer` (line 134):

```go
// AddSecurityGroup adds a security group to a server.
func (a *ComputeAPI) AddSecurityGroup(id, name string) error {
	return a.ServerAction(id, map[string]any{
		"addSecurityGroup": map[string]string{"name": name},
	})
}

// RemoveSecurityGroup removes a security group from a server.
func (a *ComputeAPI) RemoveSecurityGroup(id, name string) error {
	return a.ServerAction(id, map[string]any{
		"removeSecurityGroup": map[string]string{"name": name},
	})
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/api/ -run 'TestAddSecurityGroup|TestRemoveSecurityGroup' -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/api/compute.go internal/api/compute_test.go
git commit -m "Add AddSecurityGroup/RemoveSecurityGroup API methods (#40)"
```

---

### Task 2: Add server security group commands

**Files:**
- Create: `cmd/server/security_group.go`
- Modify: `cmd/server/server.go:35` (add commands in init)

- [ ] **Step 1: Create `cmd/server/security_group.go`**

```go
package server

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/crowdy/conoha-cli/cmd/cmdutil"
	"github.com/crowdy/conoha-cli/internal/prompt"
)

func init() {
	addSecurityGroupCmd.Flags().String("name", "", "security group name")
	addSecurityGroupCmd.MarkFlagRequired("name")

	removeSecurityGroupCmd.Flags().String("name", "", "security group name")
	removeSecurityGroupCmd.MarkFlagRequired("name")
}

var addSecurityGroupCmd = &cobra.Command{
	Use:     "add-security-group <id|name>",
	Aliases: []string{"add-sg"},
	Short:   "Add a security group to a server",
	Args:    cmdutil.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		compute, err := getComputeAPI(cmd)
		if err != nil {
			return err
		}
		id, err := resolveServerID(compute, args[0])
		if err != nil {
			return err
		}
		name, _ := cmd.Flags().GetString("name")
		ok, err := prompt.Confirm(fmt.Sprintf("Add security group %q to server %s?", name, args[0]))
		if err != nil {
			return err
		}
		if !ok {
			fmt.Fprintln(os.Stderr, "Cancelled.")
			return nil
		}
		if err := compute.AddSecurityGroup(id, name); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "Security group %q added to server %s\n", name, args[0])
		return nil
	},
}

var removeSecurityGroupCmd = &cobra.Command{
	Use:     "remove-security-group <id|name>",
	Aliases: []string{"remove-sg"},
	Short:   "Remove a security group from a server",
	Args:    cmdutil.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		compute, err := getComputeAPI(cmd)
		if err != nil {
			return err
		}
		id, err := resolveServerID(compute, args[0])
		if err != nil {
			return err
		}
		name, _ := cmd.Flags().GetString("name")
		ok, err := prompt.Confirm(fmt.Sprintf("Remove security group %q from server %s?", name, args[0]))
		if err != nil {
			return err
		}
		if !ok {
			fmt.Fprintln(os.Stderr, "Cancelled.")
			return nil
		}
		if err := compute.RemoveSecurityGroup(id, name); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "Security group %q removed from server %s\n", name, args[0])
		return nil
	},
}
```

- [ ] **Step 2: Register commands in `cmd/server/server.go`**

Add after line 35 (`Cmd.AddCommand(deployCmd)`):

```go
	Cmd.AddCommand(addSecurityGroupCmd)
	Cmd.AddCommand(removeSecurityGroupCmd)
```

- [ ] **Step 3: Verify compilation**

Run: `go build ./...`
Expected: success

- [ ] **Step 4: Commit**

```bash
git add cmd/server/security_group.go cmd/server/server.go
git commit -m "Add server add-security-group/remove-security-group commands (#40)"
```

---

### Task 3: Verify and finalize

- [ ] **Step 1: Run full test suite**

Run: `make test`
Expected: all tests pass

- [ ] **Step 2: Run linter**

Run: `make lint`
Expected: no issues

- [ ] **Step 3: Manual verification of help output**

Run: `go run . server add-security-group --help`
Expected output includes:
- Usage: `add-security-group <id|name>`
- Aliases: `add-sg`
- Required flag: `--name`

Run: `go run . server remove-security-group --help`
Expected output includes:
- Usage: `remove-security-group <id|name>`
- Aliases: `remove-sg`
- Required flag: `--name`

- [ ] **Step 4: Verify commands appear in server help**

Run: `go run . server --help`
Expected: both `add-security-group` and `remove-security-group` listed under Available Commands
