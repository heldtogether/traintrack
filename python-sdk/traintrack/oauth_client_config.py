import json
from pathlib import Path
from dataclasses import dataclass

DEFAULT_CONFIG_PATH = Path.home() / ".traintrack" / "oauth-client-config.json"


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
