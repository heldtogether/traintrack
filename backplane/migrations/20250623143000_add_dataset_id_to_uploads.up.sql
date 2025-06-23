-- Add nullable dataset_id column
ALTER TABLE uploads
ADD COLUMN dataset_id UUID;

-- Add foreign key constraint (on nullable column)
ALTER TABLE uploads
ADD CONSTRAINT uploads_dataset_id_fk
FOREIGN KEY (dataset_id) REFERENCES datasets(id)
ON DELETE SET NULL;

