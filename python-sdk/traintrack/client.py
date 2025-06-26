import os

import json
import os
from pathlib import Path
from datetime import datetime
from typing import Optional
from dataclasses import dataclass
from requests_oauthlib import OAuth2Session
from oauthlib.oauth2 import TokenExpiredError

DEFAULT_CONFIG_PATH = Path.home() / ".traintrack" / "config.json"
DEFAULT_TOKEN_PATH = Path.home() / ".traintrack" / "credentials.json"


@dataclass
class StoredConfig:
    name: str
    client_id: str
    auth_url: str

    @staticmethod
    def from_dict(data: dict) -> "StoredConfig":
        return StoredConfig(
            name=data["name"],
            client_id=data["client_id"],
            auth_url=data["auth_url"],
        )


def load_config(path: Path = DEFAULT_CONFIG_PATH) -> StoredConfig:
    try:
        with open(path, "r") as f:
            data = json.load(f)
        stored = StoredConfig.from_dict(data)
    except (OSError, json.JSONDecodeError, KeyError, ValueError) as e:
        raise RuntimeError(f"Failed to load config: {e}")

    return stored


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


class TraintrackClient:
    def __init__(self, base_url=None):
        self.base_url = base_url or os.getenv(
            "TRAINTRACK_API_URL", "http://localhost:8080"
        )

        config = load_config()
        token = load_token()

        self.session = OAuth2Session(
            client_id=config.client_id,
            token={
                "access_token": token.access_token,
                "refresh_token": token.refresh_token,
                "token_type": "Bearer",
                "expires_at": token.expiry.timestamp(),
                "id_token": token.id_token,
            },
            auto_refresh_url=config.auth_url,
            auto_refresh_kwargs={
                "client_id": config.client_id,
            },
            token_updater=save_token,
        )

    def get(self, path, **kwargs):
        return self.session.get(f"{self.base_url}{path}", **kwargs)

    def post(self, path, **kwargs):
        return self.session.post(f"{self.base_url}{path}", **kwargs)
