# GIRU JARVIS AI Assistant - Multi-stage Build
# Stage 1: Build dependencies
FROM python:3.13-slim AS builder

WORKDIR /app

# Install build dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    portaudio19-dev \
    build-essential \
    && rm -rf /var/lib/apt/lists/*

# Install Python dependencies to user site
COPY backend/requirements.txt /app/backend/requirements.txt
RUN pip install --user --no-cache-dir --upgrade pip \
    && pip install --user --no-cache-dir -r /app/backend/requirements.txt

# Stage 2: Runtime
FROM python:3.13-slim

LABEL maintainer="ASGARD Platform"
LABEL description="GIRU JARVIS AI Assistant Backend"
LABEL version="2.1"

# Install runtime dependencies only
RUN apt-get update && apt-get install -y --no-install-recommends \
    portaudio19-dev \
    espeak \
    && rm -rf /var/lib/apt/lists/* \
    && apt-get clean

# Create non-root user
RUN useradd -m -u 1000 -s /bin/bash giru

WORKDIR /app

# Copy Python packages from builder
COPY --from=builder /root/.local /home/giru/.local

# Copy backend source code
COPY --chown=giru:giru backend /app/backend

# Create data directory for SQLite database
RUN mkdir -p /app/data && chown -R giru:giru /app

# Environment configuration
ENV GIRU_PORT=7777
ENV GIRU_MONITOR_PORT=7778
ENV GIRU_DB_PATH=/app/data/giru.db
ENV PYTHONUNBUFFERED=1
ENV PYTHONDONTWRITEBYTECODE=1
ENV PATH=/home/giru/.local/bin:$PATH

# Switch to non-root user
USER giru

# Expose both main and monitor ports
EXPOSE 7777 7778

# Health check using socket (more reliable)
HEALTHCHECK --interval=30s --timeout=10s --start-period=10s --retries=3 \
    CMD python -c "import socket; s=socket.socket(); s.settimeout(5); s.connect(('127.0.0.1', 7777)); s.close()" || exit 1

# Run the server
CMD ["python", "-u", "backend/giru_server.py"]
