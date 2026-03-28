package ssh

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

// ConnectConfig holds SSH connection parameters.
type ConnectConfig struct {
	Host    string // IP or hostname
	Port    string // default "22"
	User    string // default "root"
	KeyPath string // path to private key file
}

// Connect establishes an SSH connection.
func Connect(cfg ConnectConfig) (*ssh.Client, error) {
	if cfg.Port == "" {
		cfg.Port = "22"
	}
	if cfg.User == "" {
		cfg.User = "root"
	}

	key, err := os.ReadFile(cfg.KeyPath)
	if err != nil {
		return nil, fmt.Errorf("read key %s: %w", cfg.KeyPath, err)
	}

	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return nil, fmt.Errorf("parse key %s: %w", cfg.KeyPath, err)
	}

	config := &ssh.ClientConfig{
		User:            cfg.User,
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(signer)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // personal VPS use
		Timeout:         30 * time.Second,
	}

	addr := fmt.Sprintf("%s:%s", cfg.Host, cfg.Port)
	return ssh.Dial("tcp", addr, config)
}

// RunScript uploads and executes a script on the remote server.
// Environment variables are exported before the script runs.
// Stdout/stderr are streamed to the provided writers.
// Returns the remote exit code.
//
// NOTE: env keys must be validated with ValidateEnvKey before calling.
func RunScript(client *ssh.Client, script []byte, env map[string]string, stdout, stderr io.Writer) (int, error) {
	if client == nil {
		return -1, fmt.Errorf("SSH client is nil")
	}

	session, err := client.NewSession()
	if err != nil {
		return -1, err
	}
	defer func() { _ = session.Close() }()

	session.Stdout = stdout
	session.Stderr = stderr

	var envPrefix string
	for k, v := range env {
		escaped := strings.ReplaceAll(v, "'", "'\\''")
		envPrefix += fmt.Sprintf("export %s='%s'; ", k, escaped)
	}

	session.Stdin = bytes.NewReader(script)
	cmd := envPrefix + "bash -s"

	if err := session.Run(cmd); err != nil {
		if exitErr, ok := err.(*ssh.ExitError); ok {
			return exitErr.ExitStatus(), nil
		}
		return -1, err
	}
	return 0, nil
}

// RunCommand executes a single command on the remote server.
// Stdout/stderr are streamed to the provided writers.
// Returns the remote exit code.
func RunCommand(client *ssh.Client, command string, stdout, stderr io.Writer) (int, error) {
	if client == nil {
		return -1, fmt.Errorf("SSH client is nil")
	}

	session, err := client.NewSession()
	if err != nil {
		return -1, err
	}
	defer func() { _ = session.Close() }()

	session.Stdout = stdout
	session.Stderr = stderr

	if err := session.Run(command); err != nil {
		if exitErr, ok := err.(*ssh.ExitError); ok {
			return exitErr.ExitStatus(), nil
		}
		return -1, err
	}
	return 0, nil
}
