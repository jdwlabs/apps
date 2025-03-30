FROM ghcr.io/actions/actions-runner:sha256-831a2607a2618e4b79d9323b4c72330f3861768a061c2b92a845e9d214d80e5b

# Update package list and install required dependencies
RUN apt-get update -y && apt-get install -y --no-install-recommends \
    sudo lsb-release gpg-agent software-properties-common curl jq unzip \
    libgtk2.0-0 libgtk-3-0 libgbm-dev libnotify-dev libnss3 libxss1 \
    libasound2 libxtst6 xauth xvfb \
    && rm -rf /var/lib/apt/lists/*

# Install Java (Zulu JDK 21)
RUN curl -fsSL https://cdn.azul.com/zulu/bin/zulu21.32.17-ca-jdk21.0.2-linux_amd64.deb -o zulu-jdk.deb \
    && dpkg -i zulu-jdk.deb \
    && rm zulu-jdk.deb

# Install Node.js 20 and npm
RUN curl -fsSL https://deb.nodesource.com/setup_20.x | bash - \
    && apt-get install -y nodejs \
    && npm install -g npm@latest \
    && rm -rf /var/lib/apt/lists/*

# Install Go 1.23
RUN curl -fsSL https://go.dev/dl/go1.23.linux-amd64.tar.gz -o go.tar.gz \
    && tar -C /usr/local -xzf go.tar.gz \
    && rm go.tar.gz

ENV PATH="/usr/local/go/bin:$PATH"
