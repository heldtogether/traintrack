import os
import requests

class TraintrackClient:
    def __init__(self, base_url=None):
        self.base_url = base_url or os.getenv("TRAINTRACK_API_URL", "http://localhost:8080")

    def get(self, path):
        return requests.get(f"{self.base_url}{path}")

    def post(self, path, json):
        return requests.post(f"{self.base_url}{path}", json=json)

