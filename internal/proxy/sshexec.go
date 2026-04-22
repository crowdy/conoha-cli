package proxy

import (
	"io"
	"os"

	"golang.org/x/crypto/ssh"

	internalssh "github.com/crowdy/conoha-cli/internal/ssh"
)

// SSHExecutor adapts *ssh.Client to the proxy.Executor interface by piping
// curl commands over an SSH session. stdin (when non-nil) is streamed as the
// HTTP request body (--data-binary @-).
type SSHExecutor struct {
	Client *ssh.Client
	// Stderr is where remote stderr is routed. nil defaults to os.Stderr so
	// curl/socket failures (e.g. "curl: (7) Failed to connect") reach the
	// operator. Pass io.Discard explicitly to suppress.
	Stderr io.Writer
}

// Run implements proxy.Executor.
func (e *SSHExecutor) Run(cmd string, stdin io.Reader, stdout io.Writer) error {
	stderr := e.Stderr
	if stderr == nil {
		stderr = os.Stderr
	}
	if stdin != nil {
		_, err := internalssh.RunWithStdin(e.Client, cmd, stdin, stdout, stderr)
		return err
	}
	_, err := internalssh.RunCommand(e.Client, cmd, stdout, stderr)
	return err
}
