CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE models (
    "id" UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name TEXT,
    version TEXT,
    parent UUID,
    description TEXT,
    dataset TEXT,
    config JSONB,
    metadata JSONB,
    environment JSONB,
    evaluation JSONB
);