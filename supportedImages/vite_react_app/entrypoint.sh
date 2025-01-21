#!/bin/sh
set -e

if [ -z "$(ls -A /home/vscode/code)" ]; then
    echo "Initializing code directory with default files..."
    cp -r /home/temp_code/* /home/vscode/code
    chown -R vscode:vscode /home/vscode/code
fi

exec sudo -u vscode /usr/lib/code-server/bin/code-server "$@"

# set -e is used to exist shell immidiatly if any commands its below fails
# ls -A /home/vscode/code return all files, hidden also present in code directory as string.
# -z check length of string, if string is empty then we execute block of if conditon, means if directroy is empty the output of syntax is true.
# other thing i already know....