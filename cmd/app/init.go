package app

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	internalssh "github.com/crowdy/conoha-cli/internal/ssh"
)

func init() {
	addAppFlags(initCmd)
}

var initCmd = &cobra.Command{
	Use:   "init <id|name>",
	Short: "Initialize app deployment on a server",
	Long:  "Install Docker, create git bare repo with post-receive hook for git-push deploys.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, err := connectToApp(cmd, args)
		if err != nil {
			return err
		}
		defer func() { _ = ctx.Client.Close() }()

		fmt.Fprintf(os.Stderr, "Initializing app %q on %s (%s)...\n", ctx.AppName, ctx.Server.Name, ctx.IP)

		script := generateInitScript(ctx.AppName)
		exitCode, err := internalssh.RunScript(ctx.Client, script, nil, os.Stdout, os.Stderr)
		if err != nil {
			return fmt.Errorf("init failed: %w", err)
		}
		if exitCode != 0 {
			return fmt.Errorf("init script exited with code %d", exitCode)
		}

		fmt.Fprintf(os.Stderr, "\nApp %q initialized on %s (%s).\n\n", ctx.AppName, ctx.Server.Name, ctx.IP)
		fmt.Fprintf(os.Stderr, "Add the remote and deploy:\n")
		fmt.Fprintf(os.Stderr, "  git remote add conoha %s@%s:/opt/conoha/%s.git\n", ctx.User, ctx.IP, ctx.AppName)
		fmt.Fprintf(os.Stderr, "  git push conoha main\n")

		return nil
	},
}

func generateInitScript(appName string) []byte {
	return []byte(fmt.Sprintf(`#!/bin/bash
set -euo pipefail

echo "==> Installing Docker..."
if ! command -v docker &>/dev/null; then
    curl -fsSL https://get.docker.com | sh
fi

echo "==> Installing Docker Compose plugin..."
if ! docker compose version &>/dev/null; then
    apt-get update -qq && apt-get install -y -qq docker-compose-plugin
fi

echo "==> Installing git..."
if ! command -v git &>/dev/null; then
    apt-get update -qq && apt-get install -y -qq git
fi

APP_NAME="%s"
REPO_DIR="/opt/conoha/${APP_NAME}.git"
WORK_DIR="/opt/conoha/${APP_NAME}"

echo "==> Creating directories..."
mkdir -p "$WORK_DIR"

if [ -d "$REPO_DIR" ]; then
    echo "Git repo already exists at $REPO_DIR, skipping."
else
    git init --bare "$REPO_DIR"
fi

echo "==> Installing post-receive hook..."
cat > "$REPO_DIR/hooks/post-receive" << 'HOOK'
#!/bin/bash
set -euo pipefail

APP_NAME="%s"
WORK_DIR="/opt/conoha/${APP_NAME}"
DEPLOY_BRANCH="main"

# Read pushed refs from stdin; only deploy on main branch push
while read -r oldrev newrev refname; do
    branch=$(basename "$refname")
    if [ "$branch" != "$DEPLOY_BRANCH" ]; then
        echo "Pushed to $branch — skipping deploy (only $DEPLOY_BRANCH triggers deploy)."
        continue
    fi

    echo "==> Checking out $DEPLOY_BRANCH..."
    GIT_DIR="$(dirname "$0")/.."
    git --work-tree="$WORK_DIR" --git-dir="$GIT_DIR" checkout -f "$DEPLOY_BRANCH"

    cd "$WORK_DIR"

    if [ -f docker-compose.yml ] || [ -f docker-compose.yaml ] || [ -f compose.yml ] || [ -f compose.yaml ]; then
        echo "==> Building and starting containers..."
        docker compose up -d --build --remove-orphans
        echo "==> Deploy complete!"
        docker compose ps
    else
        echo "Warning: No compose file found in $WORK_DIR"
        echo "Push a docker-compose.yml to enable auto-deploy."
    fi
done
HOOK
chmod +x "$REPO_DIR/hooks/post-receive"

echo "==> Done!"
`, appName, appName))
}
