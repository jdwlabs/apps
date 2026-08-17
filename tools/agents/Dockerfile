FROM ghcr.io/actions/actions-runner:2.336.0

# Update package list and install required dependencies
USER root
RUN apt-get clean && rm -rf /var/lib/apt/lists/* && \
    apt-get update -y && apt-get install -y --no-install-recommends \
    sudo lsb-release gpg-agent software-properties-common curl jq unzip \
    libgtk2.0-0 libgtk-3-0 libgbm-dev libnotify-dev libnss3 libxss1 \
    libasound2t64 libxtst6 xauth xvfb procps \
    && rm -rf /var/lib/apt/lists/*

# Install Zulu JDK 21 from the tar.gz file
RUN curl -fsSL https://cdn.azul.com/zulu/bin/zulu21.36.17-ca-jdk21.0.4-linux_x64.tar.gz -o zulu-jdk.tar.gz \
    && sha256sum zulu-jdk.tar.gz \
    && mkdir -p /usr/local/zulu-21 \
    && tar -xzf zulu-jdk.tar.gz --strip-components=1 -C /usr/local/zulu-21 \
    && rm zulu-jdk.tar.gz

# Install Node.js 24 LTS and pnpm
RUN curl -fsSL https://deb.nodesource.com/setup_24.x | bash - \
    && apt-get install -y --no-install-recommends nodejs \
    && corepack enable \
    && corepack prepare pnpm@latest --activate \
    && rm -rf /var/lib/apt/lists/*

# Install Go 1.26.2 (latest stable)
RUN curl -fsSL https://go.dev/dl/go1.26.2.linux-amd64.tar.gz | tar -C /usr/local -xz \
    && rm -rf /var/lib/apt/lists/*

# Install Playwright dependencies and browsers
RUN npm install -g playwright \
    && npx playwright install-deps \
    && npx playwright install chromium firefox webkit \
    && rm -rf /var/lib/apt/lists/*

# Install Cypress
RUN npm install -g cypress

# Set environment variables
ENV JAVA_HOME=/usr/local/zulu-21 \
    PATH="/usr/local/zulu-21/bin:/usr/local/go/bin:${PATH}"

WORKDIR /home/runner
USER runner

# Healthy = the runner agent's listener process is alive and polling for jobs.
# Long start period covers registration against the GitHub API on cold boot.
HEALTHCHECK --interval=60s --timeout=10s --start-period=120s --retries=3 \
  CMD pgrep -f "Runner.Listener" > /dev/null || exit 1