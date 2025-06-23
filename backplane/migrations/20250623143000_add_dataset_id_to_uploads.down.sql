ALTER TABLE uploads
DROP CONSTRAINT uploads_dataset_id_fk;

ALTER TABLE uploads
DROP COLUMN dataset_id;

