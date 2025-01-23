FROM codercom/code-server
USER root
RUN apt update && apt upgrade -y
RUN apt install golang -y
RUN useradd -m -u 1001 -d /home/vscode -s /bin/bash vscode
RUN mkdir -p /home/temp_code && chown -R vscode:vscode /home/vscode /home/temp_code
RUN echo "vscode ALL=(ALL) NOPASSWD: ALL" >> /etc/sudoers
COPY --chown=vscode:vscode ./code /home/temp_code/
COPY --chown=root:root entrypoint.sh /usr/local/bin/entrypoint.sh
RUN chmod +x /usr/local/bin/entrypoint.sh
USER vscode


RUN echo "export PATH=$PATH:$HOME/.go/bin"
RUN echo "export GOPATH=$PATH:$HOME/.go"
# Set working directory
WORKDIR /home/vscode
ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]
