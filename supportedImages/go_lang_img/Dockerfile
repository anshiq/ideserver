FROM codercom/code-server
USER root
RUN apt update && apt upgrade -y
RUN apt install golang -y
RUN apt-get clean && rm -rf /var/lib/apt/lists/*
ARG USERNAME=vscode
ARG USER_UID=1001
ARG USER_GID=$USER_UID
RUN groupadd --gid $USER_GID $USERNAME && \
    useradd --uid $USER_UID --gid $USER_GID -m -s /bin/bash $USERNAME && \
    echo "$USERNAME:vscode1234" | chpasswd && \
    adduser $USERNAME sudo && \
    # Create and set permissions for directories
    mkdir -p /home/temp_code && \
    mkdir -p /home/$USERNAME/code && \
    chown -R $USERNAME:$USERNAME /home/$USERNAME /home/temp_code && \
    chmod 755 /home/temp_code /home/$USERNAME/code

COPY --chown=$USERNAME:$USERNAME ./code /home/temp_code/
RUN chmod +x /usr/bin/code-server
WORKDIR /home/$USERNAME
