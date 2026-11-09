from __future__ import annotations

import sys


def main() -> None:
    command = sys.argv[1] if len(sys.argv) > 1 else "api"
    if command == "api":
        import uvicorn

        uvicorn.run(
            "ai_agent_service.main:create_app",
            factory=True,
            host="0.0.0.0",
            port=8019,
        )
    elif command == "migrate":
        from .config import Settings
        from .db import run_migrations

        settings = Settings.load()
        applied = run_migrations(settings.database_url, settings.migrations_dir)
        print("applied:", applied if applied else "(none pending)")
    else:
        raise SystemExit(f"unknown command: {command} (expected: api | migrate)")


if __name__ == "__main__":
    main()
