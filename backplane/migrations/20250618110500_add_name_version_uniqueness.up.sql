ALTER TABLE datasets
ADD CONSTRAINT unique_name_version UNIQUE (name, version);

