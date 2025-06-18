ALTER TABLE datasets
DROP CONSTRAINT fk_datasets_parent,
DROP COLUMN name,
DROP COLUMN version,
DROP COLUMN parent,
DROP COLUMN description;

