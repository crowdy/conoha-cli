package app

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/crowdy/conoha-cli/internal/config"
	proxypkg "github.com/crowdy/conoha-cli/internal/proxy"
)

// fakeAdmin records Upsert/Delete calls in order and lets tests program
// failure modes per-call.
type fakeAdmin struct {
	upserts   []proxypkg.UpsertRequest
	deletes   []string
	upsertErr func(req proxypkg.UpsertRequest) error
	deleteErr func(name string) error
}

func (f *fakeAdmin) Upsert(req proxypkg.UpsertRequest) (*proxypkg.Service, error) {
	f.upserts = append(f.upserts, req)
	if f.upsertErr != nil {
		if err := f.upsertErr(req); err != nil {
			return nil, err
		}
	}
	return &proxypkg.Service{Name: req.Name, Hosts: req.Hosts, Phase: "live", TLSStatus: "ready"}, nil
}

func (f *fakeAdmin) Delete(name string) error {
	f.deletes = append(f.deletes, name)
	if f.deleteErr != nil {
		return f.deleteErr(name)
	}
	return nil
}

func projectWithExpose(n int) *config.ProjectFile {
	pf := &config.ProjectFile{
		Name:  "myapp",
		Hosts: []string{"myapp.example.com"},
		Web:   config.WebSpec{Service: "web", Port: 8080},
	}
	for i := 0; i < n; i++ {
		pf.Expose = append(pf.Expose, config.ExposeBlock{
			Label:   fmt.Sprintf("aux%d", i),
			Host:    fmt.Sprintf("aux%d.example.com", i),
			Service: fmt.Sprintf("svc%d", i),
			Port:    9000 + i,
		})
	}
	return pf
}

func TestRegisterProxyServices_RootOnly(t *testing.T) {
	admin := &fakeAdmin{}
	pf := projectWithExpose(0)
	var log bytes.Buffer

	if err := registerProxyServices(admin, pf, &log); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(admin.upserts) != 1 {
		t.Fatalf("upserts = %d, want 1", len(admin.upserts))
	}
	if admin.upserts[0].Name != "myapp" {
		t.Errorf("root upsert name = %q", admin.upserts[0].Name)
	}
	if len(admin.deletes) != 0 {
		t.Errorf("unexpected deletes: %v", admin.deletes)
	}
}

func TestRegisterProxyServices_RootPlusExpose(t *testing.T) {
	admin := &fakeAdmin{}
	pf := projectWithExpose(2)
	var log bytes.Buffer

	if err := registerProxyServices(admin, pf, &log); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(admin.upserts) != 3 {
		t.Fatalf("upserts = %d, want 3 (root + 2 expose)", len(admin.upserts))
	}
	want := []string{"myapp", "myapp-aux0", "myapp-aux1"}
	for i, w := range want {
		if admin.upserts[i].Name != w {
			t.Errorf("upsert[%d].Name = %q, want %q", i, admin.upserts[i].Name, w)
		}
	}
	// Expose upserts carry only the block's host.
	if got, want := admin.upserts[1].Hosts, []string{"aux0.example.com"}; !stringSlicesEqual(got, want) {
		t.Errorf("upsert[1].Hosts = %v, want %v", got, want)
	}
	if len(admin.deletes) != 0 {
		t.Errorf("expected no deletes on happy path, got %v", admin.deletes)
	}
}

func TestRegisterProxyServices_MidFailureRollsBack(t *testing.T) {
	// Make the second expose upsert fail. Expected: earlier registrations
	// (root + first expose) are deleted in reverse so no orphans remain.
	admin := &fakeAdmin{
		upsertErr: func(req proxypkg.UpsertRequest) error {
			if req.Name == "myapp-aux1" {
				return errors.New("boom")
			}
			return nil
		},
	}
	pf := projectWithExpose(2)
	var log bytes.Buffer

	err := registerProxyServices(admin, pf, &log)
	if err == nil {
		t.Fatal("want error, got nil")
	}
	if !strings.Contains(err.Error(), "aux1") {
		t.Errorf("error should name failing block, got %q", err.Error())
	}
	// 3 upsert attempts were made (root, aux0 ok, aux1 fails).
	if len(admin.upserts) != 3 {
		t.Fatalf("upserts = %d, want 3", len(admin.upserts))
	}
	// Rollback: aux0 then root (reverse order of successful registrations).
	wantDeletes := []string{"myapp-aux0", "myapp"}
	if !stringSlicesEqual(admin.deletes, wantDeletes) {
		t.Errorf("rollback deletes = %v, want %v", admin.deletes, wantDeletes)
	}
}

func TestRegisterProxyServices_RootFailureNoRollback(t *testing.T) {
	// Root upsert fails: nothing was registered, so nothing to roll back.
	admin := &fakeAdmin{
		upsertErr: func(req proxypkg.UpsertRequest) error {
			return errors.New("denied")
		},
	}
	pf := projectWithExpose(2)
	var log bytes.Buffer

	if err := registerProxyServices(admin, pf, &log); err == nil {
		t.Fatal("want error")
	}
	if len(admin.upserts) != 1 {
		t.Errorf("want to stop after root failure, got %d upserts", len(admin.upserts))
	}
	if len(admin.deletes) != 0 {
		t.Errorf("no rollback expected, got %v", admin.deletes)
	}
}

func TestRegisterProxyServices_RollbackTolerates404(t *testing.T) {
	// Rollback must not abort on ErrNotFound — the service may already be
	// gone between our Upsert and rollback attempt.
	admin := &fakeAdmin{
		upsertErr: func(req proxypkg.UpsertRequest) error {
			if req.Name == "myapp-aux0" {
				return errors.New("boom")
			}
			return nil
		},
		deleteErr: func(name string) error {
			if name == "myapp" {
				return fmt.Errorf("wrapped: %w", proxypkg.ErrNotFound)
			}
			return nil
		},
	}
	pf := projectWithExpose(1)
	var log bytes.Buffer

	err := registerProxyServices(admin, pf, &log)
	if err == nil {
		t.Fatal("want error")
	}
	if !stringSlicesEqual(admin.deletes, []string{"myapp"}) {
		t.Errorf("delete order wrong: %v", admin.deletes)
	}
	if strings.Contains(log.String(), "warning: rollback delete myapp:") {
		t.Errorf("404 during rollback should be silent, log was: %s", log.String())
	}
}

func TestDeregisterProxyServices_AllGone(t *testing.T) {
	admin := &fakeAdmin{}
	pf := projectWithExpose(2)
	var log bytes.Buffer

	deregisterProxyServices(admin, pf, &log)

	want := []string{"myapp-aux1", "myapp-aux0", "myapp"}
	if !stringSlicesEqual(admin.deletes, want) {
		t.Errorf("deletes = %v, want %v (expose reverse then root)", admin.deletes, want)
	}
}

func TestDeregisterProxyServices_404sAreNonFatal(t *testing.T) {
	// Every delete returns ErrNotFound. Sweep must still visit every service
	// and emit no warnings.
	admin := &fakeAdmin{
		deleteErr: func(name string) error { return proxypkg.ErrNotFound },
	}
	pf := projectWithExpose(2)
	var log bytes.Buffer

	deregisterProxyServices(admin, pf, &log)

	if len(admin.deletes) != 3 {
		t.Errorf("deletes = %d, want 3", len(admin.deletes))
	}
	if strings.Contains(log.String(), "warning:") {
		t.Errorf("404 should be silent, log: %s", log.String())
	}
}

func TestDeregisterProxyServices_NonFatalError(t *testing.T) {
	// A non-404 error on one service must not stop the sweep; warnings go
	// to the log but root still gets its delete attempt.
	admin := &fakeAdmin{
		deleteErr: func(name string) error {
			if name == "myapp-aux0" {
				return errors.New("transient")
			}
			return nil
		},
	}
	pf := projectWithExpose(1)
	var log bytes.Buffer

	deregisterProxyServices(admin, pf, &log)

	if !stringSlicesEqual(admin.deletes, []string{"myapp-aux0", "myapp"}) {
		t.Errorf("root delete was skipped: %v", admin.deletes)
	}
	if !strings.Contains(log.String(), "warning: proxy delete myapp-aux0") {
		t.Errorf("warning missing for failing delete: %s", log.String())
	}
}

func TestExposeServiceName(t *testing.T) {
	if got := exposeServiceName("gitea", "dex"); got != "gitea-dex" {
		t.Errorf("got %q", got)
	}
}

func TestHealthFor_BlockOverridesRoot(t *testing.T) {
	pf := &config.ProjectFile{
		Name:   "app",
		Health: &config.HealthSpec{Path: "/root"},
	}
	block := &config.ExposeBlock{Health: &config.HealthSpec{Path: "/block"}}
	got := healthFor(pf, block)
	if got == nil || got.Path != "/block" {
		t.Errorf("got %+v, want block override", got)
	}

	pf.Health = nil
	block.Health = nil
	if got := healthFor(pf, block); got != nil {
		t.Errorf("both nil → nil, got %+v", got)
	}

	pf.Health = &config.HealthSpec{Path: "/root"}
	block.Health = nil
	got = healthFor(pf, block)
	if got == nil || got.Path != "/root" {
		t.Errorf("block nil → inherit root, got %+v", got)
	}
}

func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
