FROM python:3.13-slim

LABEL maintainer="ASGARD Platform"
LABEL description="GIRU JARVIS AI Assistant Backend"
LABEL version="2.0"

WORKDIR /app

# Install system dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    portaudio19-dev \
    build-essential \
    espeak \
    && rm -rf /var/lib/apt/lists/* \
    && apt-get clean

# Copy and install Python dependencies
COPY backend/requirements.txt /app/backend/requirements.txt
RUN pip install --no-cache-dir --upgrade pip \
    && pip install --no-cache-dir -r /app/backend/requirements.txt

# Copy backend source code
COPY backend /app/backend

# Create data directory for SQLite database
RUN mkdir -p /app/data && chmod 755 /app/data

# Environment configuration
ENV GIRU_PORT=7777
ENV GIRU_MONITOR_PORT=7778
ENV GIRU_DB_PATH=/app/data/giru.db
ENV PYTHONUNBUFFERED=1
ENV PYTHONDONTWRITEBYTECODE=1
ENV DISPLAY=:99
# Expose both main and monitor ports
EXPOSE 7777 7778

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD python -c "import requests; requests.get('http://localhost:7777', timeout=5)" || exit 1

# Run the server
CMD ["python", "-u", "backend/giru_server.py"]
