FROM python:3.12-slim AS base
WORKDIR /app
ENV PATH="/app/.venv/bin:$PATH" UV_PYTHON_DOWNLOADS=0 PYTHONUNBUFFERED=1 \
    AI_MIGRATIONS_DIR=/app/migrations/postgres
RUN python -m pip install --no-cache-dir uv==0.9.8 && useradd --uid 10001 --create-home aiagent
COPY services/ai-agent-service/pyproject.toml services/ai-agent-service/uv.lock ./
RUN uv sync --locked --no-install-project --no-dev
COPY services/ai-agent-service/src/ai_agent_service ./src/ai_agent_service
RUN uv sync --locked --no-dev --no-editable
COPY services/ai-agent-service/migrations ./migrations

FROM base AS test
RUN uv sync --locked --no-editable
COPY services/ai-agent-service/tests ./tests
USER aiagent
ENTRYPOINT ["python", "-m", "ai_agent_service"]
CMD ["api"]

FROM base AS runtime
USER aiagent
EXPOSE 8019
HEALTHCHECK --interval=30s --timeout=5s --start-period=20s --retries=3 \
    CMD ["python", "-c", "import urllib.request; urllib.request.urlopen('http://127.0.0.1:8019/health/ready', timeout=3)"]
ENTRYPOINT ["python", "-m", "ai_agent_service"]
CMD ["api"]
