from traintrack.datasets import Dataset, Datasets, list_datasets

def test_list_datasets():
    datasets = list_datasets()
    assert isinstance(datasets, Datasets)

def test_transform_dataset():
    dataset = Dataset(id="1", name="data", version="1.0.0", description="")
    new_dataset = dataset.transform(name="data", version="1.0.1", description="", transform_fn=None)
    assert new_dataset.parent == dataset.id
