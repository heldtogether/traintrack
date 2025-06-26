from setuptools import setup, find_packages

setup(
    name="traintrack-sdk",
    version="0.1.0",
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

