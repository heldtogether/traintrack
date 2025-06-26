import os
from requests_oauthlib import OAuth2Session
from traintrack.oauth_client_config import load_config as load_oauth_config
from traintrack.credentials import load_token, save_token
from traintrack.instance_config import load_config as load_instance_config


class TraintrackClient:
    def __init__(self, base_url=None):
        instance_config = load_instance_config()
        oauth_config = load_oauth_config()
        token = load_token()

        self.base_url = base_url or os.getenv("TRAINTRACK_API_URL", instance_config.url)

        self.session = OAuth2Session(
            client_id=oauth_config.client_id,
            token={
                "access_token": token.access_token,
                "refresh_token": token.refresh_token,
                "token_type": "Bearer",
                "expires_at": token.expiry.timestamp(),
                "id_token": token.id_token,
            },
            auto_refresh_url=oauth_config.auth_url,
            auto_refresh_kwargs={
                "client_id": oauth_config.client_id,
            },
            token_updater=save_token,
        )

    def get(self, path, **kwargs):
        return self.session.get(f"{self.base_url}{path}", **kwargs)

    def post(self, path, **kwargs):
        return self.session.post(f"{self.base_url}{path}", **kwargs)
