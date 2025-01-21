# Vite image setup
## build image from docker file
 docker build -t your_image_name .
## container run command
 docker run --name code-server-m   -v ./host/volume/path:/home/vscode/code -p 8080:8080      docker_file_image   --bind-addr 0.0.0.0:8080 --auth none