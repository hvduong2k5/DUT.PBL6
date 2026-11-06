FROM python:3.12-slim AS base
WORKDIR /app
ENV PATH="/app/.venv/bin:$PATH" UV_PYTHON_DOWNLOADS=0 PYTHONUNBUFFERED=1 \
    ANALYTICS_SCHEMA_DIR=/app/contracts/order/v3 ANALYTICS_MIGRATIONS=/app/migrations
RUN python -m pip install --no-cache-dir uv==0.9.8 && useradd --uid 10001 --create-home analytics
COPY services/analytics-service/pyproject.toml services/analytics-service/uv.lock ./
RUN uv sync --locked --no-install-project --no-dev
COPY services/analytics-service/src/analytics_service ./src/analytics_service
RUN uv sync --locked --no-dev --no-editable
COPY services/analytics-service/migrations ./migrations
COPY packages/events/schemas/order/v3 ./contracts/order/v3

FROM base AS test
RUN uv sync --locked --no-editable
COPY services/analytics-service/tests ./tests
COPY packages/events/examples/order/v3 ./fixtures/order/v3
COPY docs/03_api_specs/analytics-service.candidate.openapi.yaml ./contracts/analytics.openapi.yaml
USER analytics
ENTRYPOINT ["python", "-m", "analytics_service"]
CMD ["api"]

FROM base AS runtime
USER analytics
EXPOSE 8011
ENTRYPOINT ["python", "-m", "analytics_service"]
CMD ["api"]
