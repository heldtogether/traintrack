import contextlib
from .client import TraintrackClient

import tempfile
import pandas as pd
import os
import io

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

    def transform(self, name, description, version):
        artefacts = {}
        for n in self.artefacts:
            artefacts[n] = self.get_artefact(n)
        return Dataset(None, name, version, description, self.id, artefacts)
  
    def set_artefact(self, name, obj):
        if not isinstance(obj, (pd.DataFrame, str, bytes)):
            raise TypeError(f"Unsupported artefact type: {type(obj)}")
        self.artefacts[name] = obj

    def get_artefact(self, name):
        if name not in self.artefacts:
            raise KeyError(f"Artefact '{name}' not found in dataset")

        upload_id = self.artefacts[name]
        client = TraintrackClient()
        resp = client.get(f"/uploads/{upload_id}/{name}")
        resp.raise_for_status()

        content = resp.content

        content_disp = resp.headers.get("Content-Disposition", "")
        filename = None
        if "filename=" in content_disp:
            filename = content_disp.split("filename=")[-1].strip('"')
        else:
            filename = name  # fallback if header is missing

        ext = os.path.splitext(filename)[1].lower()

        if ext == ".csv":
            return pd.read_csv(io.BytesIO(content))
        elif ext == ".txt":
            return content.decode("utf-8")
        else:
            return content  # raw bytes

    def save(self, force=False):
        if self.id != None and force == False:
            raise Exception(
                    "datasets should be immutable but you're trying to modify an existing dataset")
        client = TraintrackClient()

        upload_ids = {}
        for name, obj in self.artefacts.items():
            with self._marshal_artefact(obj) as file_path:
                with open(file_path, "rb") as f:
                    ext = os.path.splitext(file_path)[1]
                    filename = f"{name}{ext}"
                    upload_resp = client.post(f"/uploads", files={name: (f"{filename}", f)})
                    upload_resp.raise_for_status()
                    upload_data = upload_resp.json()
                    upload_ids[name] = upload_data["id"]

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


