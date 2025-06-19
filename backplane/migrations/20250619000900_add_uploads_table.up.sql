CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE uploads (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    files JSONB NOT NULL
);

