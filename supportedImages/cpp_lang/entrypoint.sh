#!/bin/sh
set -e

if [ -z "$(ls -A /home/vscode/code)" ]; then
    echo "Initializing code directory with default files..."
    cp -r /home/temp_code/* /home/vscode/code
    # Ensure proper ownership of copied files
    chown -R vscode:vscode /home/vscode/code
fi

# Start code-server with any passed arguments
exec sudo -u vscode /usr/lib/code-server/bin/code-server "$@"
