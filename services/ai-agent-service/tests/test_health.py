from fastapi.testclient import TestClient

from ai_agent_service.config import Settings
from ai_agent_service.main import create_app


def test_health_ok_without_db():
    settings = Settings(database_url="postgresql://invalid:invalid@127.0.0.1:1/none")
    client = TestClient(create_app(settings))
    response = client.get("/health")
    assert response.status_code == 200
    assert response.json()["service"] == "ai-agent-service"


def test_ready_reports_db_down():
    settings = Settings(database_url="postgresql://invalid:invalid@127.0.0.1:1/none")
    client = TestClient(create_app(settings))
    response = client.get("/health/ready")
    assert response.status_code == 503
    assert response.json()["db"] == "down"
