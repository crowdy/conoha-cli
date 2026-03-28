package ssh

import (
	"bytes"
	"testing"
)

func TestConnectMissingKey(t *testing.T) {
	_, err := Connect(ConnectConfig{
		Host:    "127.0.0.1",
		Port:    "22",
		User:    "root",
		KeyPath: "/nonexistent/key",
	})
	if err == nil {
		t.Fatal("expected error for missing key file")
	}
}

func TestConnectDefaults(t *testing.T) {
	_, err := Connect(ConnectConfig{
		Host:    "192.0.2.1",
		KeyPath: "/nonexistent/key",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("read key")) {
		t.Errorf("expected 'read key' error, got: %v", err)
	}
}

func TestRunScriptNilClient(t *testing.T) {
	_, err := RunScript(nil, []byte("echo hi"), nil, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil {
		t.Fatal("expected error for nil client")
	}
}

func TestRunCommandNilClient(t *testing.T) {
	_, err := RunCommand(nil, "echo hi", &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil {
		t.Fatal("expected error for nil client")
	}
}

func TestRunWithStdinNilClient(t *testing.T) {
	_, err := RunWithStdin(nil, "cat", &bytes.Buffer{}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil {
		t.Fatal("expected error for nil client")
	}
}
