import contextlib
import inspect
import joblib
import os
import pickle
import platform
import subprocess
import sys
import tempfile
from .client import TraintrackClient

class Model:
    def __init__(self, id, name, version, description, parent=None, dataset=None, config=None, artefacts=None, metadata=None, environment=None, evaluation=None):
        self.id = id
        self.name = name
        self.version = version
        self.description = description
        self.parent = parent

        self.dataset = dataset
        self.config = config or {}
        self.model_obj = None
        self.metadata = metadata or {}
        self.environment = environment or {}
        self.evaluation = evaluation or None
        self.artefacts = artefacts or {}

        self._trained_model = None

    @property
    def trained_model(self):
        if self._trained_model is None and "trained_model" in self.artefacts:
            self._trained_model = self._load_trained_model()
        return self._trained_model

    def setup(self, setup_fn):
        """
        Accepts a function that returns an untrained model/class instance.
        E.g., setup_fn(dataset, config) -> model_obj
        """
        self.model_obj = setup_fn(self.dataset, self.config)
        self.metadata['setup_fn_source'] = get_fn_source(setup_fn)
        self.metadata['model_class'] = type(self.model_obj).__name__
        if hasattr(self.model_obj, 'get_params'):
            self.metadata['init_params'] = self.model_obj.get_params()

    def train(self, train_fn):
        """
        Accepts a function that takes (model_obj, dataset) and returns a trained model.
        E.g., train_fn(model_obj, dataset) -> trained_model
        """
        self._trained_model = train_fn(self.model_obj, self.dataset)
        self.metadata['train_fn_source'] = get_fn_source(train_fn)
        self.environment['runtime'] = platform.python_implementation()
        self.environment['runtime_version'] = sys.version
        self.environment['package_manager'] = detect_package_manager()
        self.environment['dependencies'] = subprocess.check_output(["pip", "freeze"]).decode()

    def eval(self, eval_fn):
        """
        Accepts a function that takes (model_obj, dataset) and returns performance metrics.
        E.g., eval_fn(model_obj, dataset) -> dict
        """
        self.evaluation = eval_fn(self.model_obj, self.dataset)
        self.metadata['eval_fn_source'] = get_fn_source(eval_fn)

    def save(self, force=False):
        """Save model and metadata."""
        if self.id != None and force == False:
            raise Exception(
                    "datasets should be immutable but you're trying to modify an existing dataset")
        client = TraintrackClient()

        upload_ids = {}
        with self._marshal_model(self._trained_model) as file_path:
            with open(file_path, "rb") as f:
                ext = os.path.splitext(file_path)[1]
                filename = f"model{ext}"
                upload_resp = client.post(f"/uploads", files={"trained_model": (f"{filename}", f)})
                upload_resp.raise_for_status()
                upload_data = upload_resp.json()
                upload_ids["model"] = upload_data["id"]

        data = {
                "name": self.name,
                "version": self.version,
                "description": self.description,
                "parent": self.parent,
                "config": self.config,
                "metadata": self.metadata,
                "environment": self.environment,
                "evaluation": self.evaluation,
                "dataset": self.dataset.id,
                "artefacts": upload_ids,
                }

        resp = client.post("/models", json=data)
        resp.raise_for_status()
        return Model(**resp.json())

    @contextlib.contextmanager
    def _marshal_model(self, obj):
        try:
            import sklearn.base
            is_sklearn = isinstance(obj, sklearn.base.BaseEstimator)
            suffix = ".joblib"
        except ImportError:
            is_sklearn = False
            suffix = ".pkl"

        with tempfile.NamedTemporaryFile(delete=False, suffix=suffix, mode="wb") as tmp:
            try:
                if is_sklearn:
                    joblib.dump(obj, tmp)
                else:
                    pickle.dump(obj, tmp)
                tmp.flush()
                yield tmp.name
            finally:
                os.unlink(tmp.name)
    
    def _load_trained_model(self):
        artefact_id = self.artefacts.get("trained_model")
        if not artefact_id:
            raise ValueError("No trained model artefact found.")

        client = TraintrackClient()
        resp = client.get(f"/uploads/{artefact_id}/trained_model")
        resp.raise_for_status()

        # Infer file extension from headers (e.g., Content-Disposition) or default to .pkl
        content_disp = resp.headers.get("Content-Disposition", "")
        ext = ".pkl"
        if "filename=" in content_disp:
            filename = content_disp.split("filename=")[1].strip('"')
            ext = os.path.splitext(filename)[1]

        with tempfile.NamedTemporaryFile(delete=False, suffix=ext) as tmp:
            tmp.write(resp.content)
            tmp.flush()
            path = tmp.name

        try:
            with open(path, "rb") as f:
                if ext == ".joblib":
                    import joblib
                    return joblib.load(f)
                else:
                    import pickle
                    return pickle.load(f)
        finally:
            os.unlink(path)

def get_fn_source(obj):
    return inspect.getsource(obj)

def detect_package_manager():
    if os.path.exists('pdm.lock'):
        return 'pdm'
    elif os.path.exists('poetry.lock'):
        return 'poetry'
    elif os.path.exists('Pipfile.lock'):
        return 'pipenv'
    elif os.path.exists('environment.yml'):
        return 'conda'
    elif os.path.exists('requirements.txt'):
        return 'pip'
    else:
        # Environment variable hints
        if os.environ.get('PDM_PROJECT_ROOT'):
            return 'pdm'
        if os.environ.get('CONDA_DEFAULT_ENV'):
            return 'conda'
        # Default fallback
        return 'unknown'

class Models:
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
        return f"<Models {len(self.items)} items>"


def list_models(client=None):
    client = client or TraintrackClient()
    resp = client.get("/models")
    resp.raise_for_status()
    items = [Model(**d) for d in resp.json()]
    return Models(items)