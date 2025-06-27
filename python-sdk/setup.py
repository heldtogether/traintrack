from setuptools import setup, find_packages
from pathlib import Path

this_dir = Path(__file__).parent
readme = this_dir / "README.md"
long_description = readme.read_text() if readme.exists() else ""

setup(
    name="traintrack-mlops",
    version="0.0.1.post0",
    description="Traintrack is a modular MLOps platform designed to manage and monitor the full lifecycle of machine learning workflows — from dataset tracking to deployment — with a robust Go-based backend and an easy-to-use Python SDK.",
    long_description=long_description,
    long_description_content_type="text/markdown",
    packages=find_packages(),
    install_requires=[
        "requests",
        "pandas",
        "pytest",
        "joblib",
        "requests_oauthlib",
        "oauthlib"
    ],
)

