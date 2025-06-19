import contextlib
from .client import TraintrackClient

import tempfile
import pandas as pd
import os

class Dataset:
    def __init__(self, id, name, version, description, parent=None, artefacts=None):
        self.id = id
        self.name = name
        self.version = version
        self.description = description
        self.parent = parent
        self.artefacts = artefacts or {}

    def __repr__(self):
        return f"<Dataset {self.name}:{self.version}>"

    def transform(self, name, description, version, transform_fn):
        # new_df = transform_fn(df)
        return Dataset(None, name, version, description, self.id)

    def save(self, force=False):
        if self.id != None and force == False:
            raise Exception(
                "datasets should be immutable but you're trying to modify an existing dataset")
        client = TraintrackClient()

        upload_ids = []
        for name, obj in self.artefacts.items():
            with self._marshal_artefact(obj) as file_path:
                with open(file_path, "rb") as f:
                    ext = os.path.splitext(file_path)[1]
                    filename = f"{name}{ext}"
                    upload_resp = client.post(f"/uploads", files={"files": (f"{filename}", f)})
                    upload_resp.raise_for_status()
                    upload_data = upload_resp.json()
                    upload_ids.append(upload_data["id"])

        data = {
            "name": self.name,
            "version": self.version,
            "description": self.description,
            "parent": self.parent,
            "artefacts": upload_ids,
        }
        resp = client.post("/datasets", json=data)
        resp.raise_for_status()
        return Dataset(**resp.json())

    def set_artefact(self, name, obj):
        if not isinstance(obj, (pd.DataFrame, str, bytes)):
            raise TypeError(f"Unsupported artefact type: {type(obj)}")
        self.artefacts[name] = obj

    @contextlib.contextmanager
    def _marshal_artefact(self, obj):
        if isinstance(obj, pd.DataFrame):
            with tempfile.NamedTemporaryFile(delete=False, suffix=".csv", mode="w") as tmp:
                obj.to_csv(tmp.name, index=False)
                tmp.flush()
                yield tmp.name
                os.unlink(tmp.name)
        elif isinstance(obj, str):
            with tempfile.NamedTemporaryFile(delete=False, suffix=".txt", mode="w") as tmp:
                tmp.write(obj)
                tmp.flush()
                yield tmp.name
                os.unlink(tmp.name)
        elif isinstance(obj, bytes):
            with tempfile.NamedTemporaryFile(delete=False, suffix=".bin", mode="wb") as tmp:
                tmp.write(obj)
                tmp.flush()
                yield tmp.name
                os.unlink(tmp.name)
        else:
            raise TypeError(f"Unsupported artefact type: {type(obj)}")


class Datasets:
    def __init__(self, items):
        self.items = items

    def __iter__(self):
        return iter(self.items)

    def __getitem__(self, i):
        return self.items[i]

    def __len__(self):
        return len(self.items)

    def filter_by_name(self, name):
        return [d for d in self.items if d.name == name]

    def latest_version(self, name):
        # Very naive version parser: assumes semantic versioning and sorts lexically
        versions = [d for d in self.items if d.name == name]
        if not versions:
            return None
        return sorted(versions, key=lambda d: d.version, reverse=True)[0]

    def __repr__(self):
        return f"<Datasets {len(self.items)} items>"


def list_datasets(client=None):
    client = client or TraintrackClient()
    resp = client.get("/datasets")
    resp.raise_for_status()
    items = [Dataset(**d) for d in resp.json()]
    return Datasets(items)


