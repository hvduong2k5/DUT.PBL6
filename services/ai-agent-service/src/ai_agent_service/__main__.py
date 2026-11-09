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
    elif command == "cleanup":
        # Retention job: delete conversations past their retention window
        # (30 days logged-in / 15 days guest, set at creation time).
        from datetime import UTC, datetime

        from .config import Settings
        from .platform.conversations import PostgresConversationStore

        settings = Settings.load()
        if settings.fixtures_mode:
            print("fixtures mode: nothing to clean")
            return
        store = PostgresConversationStore(settings.database_url)
        deleted = store.cleanup_expired(datetime.now(UTC))
        print(f"deleted expired conversations: {deleted}")
    else:
        raise SystemExit(
            f"unknown command: {command} (expected: api | migrate | cleanup)"
        )


if __name__ == "__main__":
    main()
