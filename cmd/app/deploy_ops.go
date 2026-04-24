package app

import (
	"bytes"
	"fmt"
	"io"
	"os"

	"golang.org/x/crypto/ssh"

	proxypkg "github.com/crowdy/conoha-cli/internal/proxy"
	internalssh "github.com/crowdy/conoha-cli/internal/ssh"
)

// DeployOps abstracts every side-effecting dependency that runProxyDeploy
// needs: SSH command execution (bytes in, bytes out) and proxy Admin API
// calls. Separates the state machine from the transport so it can be unit-
// tested with a fake.
//
// Semantics:
//   - Run: execute cmd on the remote. Returns exit code + stdout bytes.
//     stderr is forwarded to the local process stderr (so operators see
//     `docker compose` output in real time). A non-nil Go error is a
//     transport-level failure (SSH auth, channel close, etc.), distinct
//     from a non-zero exit code.
//   - RunStream: like Run but streams stdout to the caller's writer
//     without buffering. Used for upload paths where stdout isn't inspected.
//   - Proxy: the Admin API client. Injected rather than constructed so
//     tests can swap it without setting up a fake Executor.
type DeployOps interface {
	Run(cmd string, stdin io.Reader) (exitCode int, stdout []byte, err error)
	RunStream(cmd string, stdin io.Reader, stdout io.Writer) (exitCode int, err error)
	Proxy() DeployProxyAPI
}

// DeployProxyAPI is the subset of proxypkg.Client that runProxyDeploy uses.
// Kept tight so fakes don't drift. Rollback is wired in for phase 3's
// reverse-rollback-on-partial-failure path.
type DeployProxyAPI interface {
	Get(name string) (*proxypkg.Service, error)
	Deploy(name string, req proxypkg.DeployRequest) (*proxypkg.Service, error)
	Rollback(name string, drainMs int) (*proxypkg.Service, error)
}

// sshDeployOps is the production DeployOps: wraps a live *ssh.Client +
// proxypkg.Client.
type sshDeployOps struct {
	client *ssh.Client
	admin  *proxypkg.Client
}

func newSSHDeployOps(client *ssh.Client, admin *proxypkg.Client) *sshDeployOps {
	return &sshDeployOps{client: client, admin: admin}
}

func (o *sshDeployOps) Run(cmd string, stdin io.Reader) (int, []byte, error) {
	var buf bytes.Buffer
	var code int
	var err error
	if stdin != nil {
		code, err = internalssh.RunWithStdin(o.client, cmd, stdin, &buf, os.Stderr)
	} else {
		code, err = internalssh.RunCommand(o.client, cmd, &buf, os.Stderr)
	}
	return code, buf.Bytes(), err
}

func (o *sshDeployOps) RunStream(cmd string, stdin io.Reader, stdout io.Writer) (int, error) {
	if stdin != nil {
		return internalssh.RunWithStdin(o.client, cmd, stdin, stdout, os.Stderr)
	}
	return internalssh.RunCommand(o.client, cmd, stdout, os.Stderr)
}

func (o *sshDeployOps) Proxy() DeployProxyAPI { return o.admin }

// runRemoteOps is the ops-side analog of runRemote: returns a typed error
// when the remote exit is non-zero, otherwise the transport error. Streams
// stdout to the local process.
func runRemoteOps(ops DeployOps, command string, stdin io.Reader) error {
	code, err := ops.RunStream(command, stdin, os.Stdout)
	if err != nil {
		return err
	}
	if code != 0 {
		return fmt.Errorf("remote exit %d", code)
	}
	return nil
}

// tearDownSlotOps is the ops-side analog of tearDownSlot: best-effort,
// discards the exit code + output.
func tearDownSlotOps(ops DeployOps, app, slot string) {
	work := fmt.Sprintf("/opt/conoha/%s/%s", app, slot)
	cmd := fmt.Sprintf(
		"docker compose -p %s -f '%s/conoha-override.yml' down 2>/dev/null || true; rm -rf '%s' || true",
		slotProjectName(app, slot), work, work)
	_, _, _ = ops.Run(cmd, nil)
}
