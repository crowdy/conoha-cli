package cmdutil

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/spf13/cobra"
)

const (
	DefaultWaitInterval = 10 * time.Second
	DefaultWaitTimeout  = 5 * time.Minute
)

// WaitConfig holds parameters for the polling loop.
type WaitConfig struct {
	Interval time.Duration
	Timeout  time.Duration
	Resource string // e.g. "server my-server"
}

// WaitFor polls checkFn until it returns done=true or an error occurs.
// checkFn returns (done, status, error):
//   - done=true:  success, stop polling
//   - done=false: keep polling, print status
//   - err!=nil:   abort with error
func WaitFor(cfg WaitConfig, checkFn func() (done bool, status string, err error)) error {
	if cfg.Interval == 0 {
		cfg.Interval = DefaultWaitInterval
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = DefaultWaitTimeout
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	deadline := time.Now().Add(cfg.Timeout)
	for {
		done, status, err := checkFn()
		if err != nil {
			return err
		}
		if done {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("timeout waiting for %s (status: %s)", cfg.Resource, status)
		}
		if status != "" {
			fmt.Fprintf(os.Stderr, "  %s status: %s\n", cfg.Resource, status)
		}

		select {
		case <-ctx.Done():
			fmt.Fprintf(os.Stderr, "\nInterrupted. Operation is still in progress on the server.\n")
			return fmt.Errorf("interrupted while waiting for %s", cfg.Resource)
		case <-time.After(cfg.Interval):
		}
	}
}

// AddWaitFlags adds --wait and --wait-timeout flags to a command.
func AddWaitFlags(cmd *cobra.Command) {
	cmd.Flags().Bool("wait", false, "wait for operation to complete")
	cmd.Flags().Duration("wait-timeout", DefaultWaitTimeout, "maximum wait time")
}

// GetWaitConfig reads --wait and --wait-timeout flags from the command.
// Returns nil if --wait is not set.
func GetWaitConfig(cmd *cobra.Command, resource string) *WaitConfig {
	wait, _ := cmd.Flags().GetBool("wait")
	if !wait {
		return nil
	}
	timeout, _ := cmd.Flags().GetDuration("wait-timeout")
	return &WaitConfig{
		Timeout:  timeout,
		Resource: resource,
	}
}
