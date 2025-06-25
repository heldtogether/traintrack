ALTER TABLE models
ADD CONSTRAINT models_unique_name_version UNIQUE (name, version);