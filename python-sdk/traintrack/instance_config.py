import json
import logging
import requests
from datetime import datetime, timedelta, timezone
from dataclasses import dataclass, asdict
from pathlib import Path
from urllib.parse import urljoin
from traintrack.oauth_client_config import (
    StoredConfig,
    DEFAULT_CONFIG_PATH as DEFAULT_AUTH_CONFIG_PATH,
)

DEFAULT_CONFIG_PATH = Path.home() / ".traintrack" / "instance-config.json"
REFRESH_INTERVAL = timedelta(hours=4)


@dataclass
class InstanceConfig:
    url: str
    last_fetched: str = None  # ISO 8601 format

    def refresh_auth_config(self) -> "InstanceConfig":
        last_fetched_time = (
            datetime.fromisoformat(self.last_fetched.replace("Z", "+00:00"))
            if self.last_fetched
            else datetime.fromtimestamp(0, tz=timezone.utc)
        )

        if datetime.now(timezone.utc) - last_fetched_time >= REFRESH_INTERVAL:
            try:
                full_url = urljoin(
                    self.url.rstrip("/") + "/", ".well-known/oauth-client-config"
                )
                resp = requests.get(full_url, timeout=5)

                if resp.status_code != 200:
                    logging.warning(
                        f"Unable to refresh oauth client config: {resp.status_code} - {resp.reason}"
                    )
                    return self

                data = resp.json()
                auth_config = StoredConfig.from_dict(data)

                # Save new auth config
                DEFAULT_AUTH_CONFIG_PATH.parent.mkdir(parents=True, exist_ok=True)
                with open(DEFAULT_AUTH_CONFIG_PATH, "w") as f:
                    json.dump(asdict(auth_config), f, indent=2)

                # Update instance config with new fetch time
                self.last_fetched = (
                    datetime.now(timezone.utc)
                    .isoformat(timespec="microseconds")
                    .replace("+00:00", "Z")
                )
                DEFAULT_CONFIG_PATH.parent.mkdir(parents=True, exist_ok=True)
                with open(DEFAULT_CONFIG_PATH, "w") as f:
                    json.dump(asdict(self), f, indent=2)

            except Exception as e:
                logging.warning(f"Unable to refresh oauth client config: {e}")

        return self


def load_config(path: Path = DEFAULT_CONFIG_PATH) -> InstanceConfig:
    with path.open("r", encoding="utf-8") as f:
        data = json.load(f)
    config = InstanceConfig(**data)
    return config.refresh_auth_config()
