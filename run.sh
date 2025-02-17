#!/bin/bash


# Start the frontend development server in the background.
echo "Starting frontend development server..."
(
  cd src/frontend || { echo "Directory src/frontend not found"; exit 1; }
  npm run dev
) &

# Start the API service development server in the background.
echo "Starting API service development server..."
(
  cd src/apiservice || { echo "Directory src/apiservice not found"; exit 1; }
  npm run dev
) &

# Start autossh tunnel in the background.
# echo "Starting autossh tunnel..."
# autossh -M 0 -i "awssubject.pem" -N \
#   -R 3000:localhost:3000 \
#   -R 8080:localhost:8080 \
#   ubuntu@ec2-18-235-57-141.compute-1.amazonaws.com &

# # Optionally, capture autossh PID if you want to kill it later.
# AUTOSSH_PID=$!
# echo "Autossh PID: $AUTOSSH_PID"


# Wait for all background processes to exit.
wait
