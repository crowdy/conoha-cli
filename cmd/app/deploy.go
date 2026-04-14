package app

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/spf13/cobra"

	clerrors "github.com/crowdy/conoha-cli/internal/errors"
	internalssh "github.com/crowdy/conoha-cli/internal/ssh"
)

func init() {
	addAppFlags(deployCmd)
	deployCmd.Flags().StringP("compose-file", "f", "", "compose file path (auto-detected if not specified)")
}

var deployCmd = &cobra.Command{
	Use:   "deploy <id|name>",
	Short: "Deploy current directory to a server",
	Long:  "Archive current directory, upload via SSH, and run docker compose up.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, err := connectToApp(cmd, args)
		if err != nil {
			return err
		}
		defer func() { _ = ctx.Client.Close() }()
		return deployApp(ctx)
	},
}

// composeFileRegex allows alphanumeric, hyphens, dots, underscores, and path separators.
var composeFileRegex = regexp.MustCompile(`^[a-zA-Z0-9/][a-zA-Z0-9._/-]*$`)

// validateComposeFilePath checks that the path contains only safe characters.
func validateComposeFilePath(path string) error {
	if !composeFileRegex.MatchString(path) {
		return &clerrors.ValidationError{
			Field:   "compose-file",
			Message: fmt.Sprintf("invalid compose file path %q: must contain only alphanumeric, hyphens, dots, underscores, and slashes", path),
		}
	}
	return nil
}

// resolveComposeFile returns the compose file to use.
// If explicit is non-empty, it validates that the file exists.
// Otherwise it auto-detects using the priority order.
func resolveComposeFile(explicit string) (string, error) {
	if explicit != "" {
		if err := validateComposeFilePath(explicit); err != nil {
			return "", err
		}
		if _, err := os.Stat(explicit); err != nil {
			return "", &clerrors.ValidationError{
				Field:   "compose-file",
				Message: fmt.Sprintf("compose file not found: %s", explicit),
			}
		}
		return explicit, nil
	}
	return detectComposeFile(".")
}

func deployApp(ctx *appContext) error {
	// Resolve compose file
	composeFile, err := resolveComposeFile(ctx.ComposeFile)
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "Using compose file: %s\n", composeFile)

	// Load .dockerignore
	patterns, err := loadIgnorePatterns(".")
	if err != nil {
		return err
	}

	// Create tar.gz
	fmt.Fprintf(os.Stderr, "Archiving current directory...\n")
	var buf bytes.Buffer
	if err := createTarGz(".", patterns, &buf); err != nil {
		return fmt.Errorf("create archive: %w", err)
	}
	fmt.Fprintf(os.Stderr, "Uploading to %s (%s)...\n", ctx.Server.Name, ctx.IP)

	// Upload: backup existing .env, clean-extract archive, then restore .env from
	// .env.server (if present in archive) or from the backup.
	workDir := "/opt/conoha/" + ctx.AppName
	uploadCmd := buildUploadCmd(workDir)
	exitCode, err := internalssh.RunWithStdin(ctx.Client, uploadCmd, &buf, os.Stdout, os.Stderr)
	if err != nil {
		return fmt.Errorf("upload failed: %w", err)
	}
	if exitCode != 0 {
		return fmt.Errorf("upload exited with code %d", exitCode)
	}

	// Run docker compose (passes --env-file when .env exists for build-time vars)
	fmt.Fprintf(os.Stderr, "Building and starting containers...\n")
	composeCmd := buildComposeCmd(workDir, composeFile)
	exitCode, err = internalssh.RunCommand(ctx.Client, composeCmd, os.Stdout, os.Stderr)
	if err != nil {
		return fmt.Errorf("deploy failed: %w", err)
	}
	if exitCode != 0 {
		return fmt.Errorf("deploy exited with code %d", exitCode)
	}

	fmt.Fprintf(os.Stderr, "Deploy complete.\n")
	return nil
}

// buildUploadCmd generates the shell command that uploads the tar archive.
// It preserves any pre-existing .env on the server:
//   - If the new archive contains .env.server, it is copied to .env (fixes #81).
//   - Otherwise the pre-existing .env is restored from a backup (fixes #85).
func buildUploadCmd(workDir string) string {
	return fmt.Sprintf(
		"ENV_BACKUP=$([ -f '%[1]s/.env' ] && cat '%[1]s/.env' || true) && "+
			"rm -rf '%[1]s' && mkdir -p '%[1]s' && tar xzf - -C '%[1]s' && "+
			"{ if [ -f '%[1]s/.env.server' ]; then "+
			"cp '%[1]s/.env.server' '%[1]s/.env'; "+
			"elif [ -n \"$ENV_BACKUP\" ]; then "+
			"printf '%%s' \"$ENV_BACKUP\" > '%[1]s/.env'; "+
			"fi; }",
		workDir)
}

// buildComposeCmd generates the shell command that runs docker compose.
// When .env exists, --env-file is passed so variables are substituted in
// compose.yml including build.args, making them available at Docker build time (#82).
func buildComposeCmd(workDir, composeFile string) string {
	return fmt.Sprintf(
		"cd '%[1]s' && "+
			"{ if [ -f .env ]; then "+
			"docker compose --env-file .env -f '%[2]s' up -d --build --remove-orphans; "+
			"else "+
			"docker compose -f '%[2]s' up -d --build --remove-orphans; "+
			"fi; } && "+
			"docker compose -f '%[2]s' ps",
		workDir, composeFile)
}

// composeFileNames lists compose files in detection priority order.
var composeFileNames = []string{
	"conoha-docker-compose.yml",
	"conoha-docker-compose.yaml",
	"docker-compose.yml",
	"docker-compose.yaml",
	"compose.yml",
	"compose.yaml",
}

// detectComposeFile returns the first compose file found in dir.
func detectComposeFile(dir string) (string, error) {
	for _, name := range composeFileNames {
		if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
			return name, nil
		}
	}
	return "", &clerrors.ValidationError{
		Field:   "compose-file",
		Message: "no compose file found in current directory (checked conoha-docker-compose.yml/yaml, docker-compose.yml/yaml, compose.yml/yaml)",
	}
}
