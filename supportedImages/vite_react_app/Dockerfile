FROM codercom/code-server:4.96.4

# Switch to root to perform installations
USER root

# Update and install dependencies in a single layer
RUN apt-get update && \
    apt-get upgrade -y && \
    apt-get install -y \
    nodejs \
    npm \
    sudo && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

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
