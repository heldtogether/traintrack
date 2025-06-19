import os
import requests


class TraintrackClient:
    def __init__(self, base_url=None):
        self.base_url = base_url or os.getenv(
            "TRAINTRACK_API_URL", "http://localhost:8080")

    def get(self, path, **kwargs):
        return requests.get(f"{self.base_url}{path}", **kwargs)

    def post(self, path, **kwargs):
        return requests.post(f"{self.base_url}{path}", **kwargs)
