from setuptools import setup, find_packages

setup(
    name="traintrack-mlops",
    version="0.1.0",
    description="Traintrack is a modular MLOps platform designed to manage and monitor the full lifecycle of machine learning workflows — from dataset tracking to deployment — with a robust Go-based backend and an easy-to-use Python SDK.",
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

