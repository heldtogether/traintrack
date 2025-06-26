import json
import os
from pathlib import Path
from datetime import datetime
from typing import Optional
from dataclasses import dataclass

DEFAULT_TOKEN_PATH = Path.home() / ".traintrack" / "credentials.json"


@dataclass
class StoredToken:
    access_token: str
    refresh_token: str
    id_token: Optional[str]
    expiry: datetime

    @staticmethod
    def from_dict(data: dict) -> "StoredToken":
        return StoredToken(
            access_token=data["access_token"],
            refresh_token=data["refresh_token"],
            id_token=data.get("id_token"),
            expiry=datetime.fromisoformat(data["expiry"]),
        )


def load_token(path: Path = DEFAULT_TOKEN_PATH) -> StoredToken:
    try:
        with open(path, "r") as f:
            data = json.load(f)
        stored = StoredToken.from_dict(data)
    except (OSError, json.JSONDecodeError, KeyError, ValueError) as e:
        raise RuntimeError(f"Failed to load token: {e}")

    return stored


def save_token(token, path=DEFAULT_TOKEN_PATH):
    os.makedirs(path.parent, exist_ok=True)
    with open(path, "w") as f:
        json.dump(
            {
                "access_token": token["access_token"],
                "refresh_token": token["refresh_token"],
                "id_token": token.get("id_token"),
                "expiry": datetime.utcfromtimestamp(token["expires_at"]).isoformat(),
            },
            f,
            indent=2,
        )
