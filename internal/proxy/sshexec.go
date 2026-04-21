package proxy

import (
	"io"

	"golang.org/x/crypto/ssh"

	internalssh "github.com/crowdy/conoha-cli/internal/ssh"
)

// SSHExecutor adapts *ssh.Client to the proxy.Executor interface by piping
// curl commands over an SSH session. stdin (when non-nil) is streamed as the
// HTTP request body (--data-binary @-).
type SSHExecutor struct {
	Client *ssh.Client
	Stderr io.Writer // where SSH stderr gets routed (can be nil → discarded)
}

// Run implements proxy.Executor.
func (e *SSHExecutor) Run(cmd string, stdin io.Reader, stdout io.Writer) error {
	stderr := e.Stderr
	if stderr == nil {
		stderr = io.Discard
	}
	if stdin != nil {
		_, err := internalssh.RunWithStdin(e.Client, cmd, stdin, stdout, stderr)
		return err
	}
	_, err := internalssh.RunCommand(e.Client, cmd, stdout, stderr)
	return err
}
