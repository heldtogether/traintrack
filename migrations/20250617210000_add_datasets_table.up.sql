CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE datasets( 
	"id" UUID PRIMARY KEY DEFAULT uuid_generate_v4()
);

