from .client import TraintrackClient

class Dataset:
    def __init__(self, id, name, version, description, parent=None):
        self.id = id
        self.name = name
        self.version = version
        self.description = description
        self.parent = parent

    def __repr__(self):
        return f"<Dataset {self.name}:{self.version}>"

    def transform(self, name, description, version, transform_fn):
        # new_df = transform_fn(df)
        return Dataset(None, name, version, description, self.id)

    def save(self, force=False):
        if self.id != None and force == False:
            raise Exception("datasets should be immutable but you're trying to modify an existing dataset")
        client = TraintrackClient()
        data = {
            "name": self.name,
            "version": self.version,
            "description": self.description,
            "parent": self.parent,
        }
        resp = client.post("/datasets", json=data)
        resp.raise_for_status()
        return Dataset(**resp.json())



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

'''
def create_dataset(name, version, description, client=None):
    client = client or TraintrackClient()
    data = {
        "name": name,
        "version": version,
        "description": description,
    }
    resp = client.post("/datasets", json=data)
    resp.raise_for_status()
    return Dataset(**resp.json())
'''
