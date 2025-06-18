ALTER TABLE datasets
ADD COLUMN name TEXT,
ADD COLUMN version TEXT,
ADD COLUMN parent UUID,
ADD COLUMN description TEXT,
ADD CONSTRAINT fk_datasets_parent
  FOREIGN KEY (parent) REFERENCES datasets(id) ON DELETE SET NULL;

