FROM ghcr.io/actions/actions-runner:2.323.0@sha256:831a2607a2618e4b79d9323b4c72330f3861768a061c2b92a845e9d214d80e5b

# Update package list and install required dependencies
USER root
RUN apt-get clean && rm -rf /var/lib/apt/lists/* && \
    apt-get update -y && apt-get install -y --no-install-recommends \
    sudo lsb-release gpg-agent software-properties-common curl jq unzip \
    libgtk2.0-0 libgtk-3-0 libgbm-dev libnotify-dev libnss3 libxss1 \
    libasound2 libxtst6 xauth xvfb \
    && rm -rf /var/lib/apt/lists/*

# Install Zulu JDK 21 from the tar.gz file
RUN curl -fsSL https://cdn.azul.com/zulu/bin/zulu21.36.17-ca-jdk21.0.4-linux_x64.tar.gz -o zulu-jdk.tar.gz \
    && sha256sum zulu-jdk.tar.gz \
    && mkdir -p /usr/local/zulu-21 \
    && tar -xzvf zulu-jdk.tar.gz --strip-components=1 -C /usr/local/zulu-21 \
    && rm zulu-jdk.tar.gz

# Install Node.js 22 and npm
RUN curl -fsSL https://deb.nodesource.com/setup_22.x | bash - \
    && apt-get install -y nodejs \
    && npm install -g npm@latest \
    && rm -rf /var/lib/apt/lists/*

# Install Go 1.24
RUN curl -fsSL https://go.dev/dl/go1.24.2.linux-amd64.tar.gz | tar -C /usr/local -xz \
    && rm -rf /var/lib/apt/lists/*

# Install Cypress
RUN npm install -g cypress

# Set environment variables
ENV JAVA_HOME=/usr/local/zulu-21 \
    PATH="/usr/local/zulu-21/bin:/usr/local/go/bin:${PATH}"
