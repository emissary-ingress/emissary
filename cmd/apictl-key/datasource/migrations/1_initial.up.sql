CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE aes_domains (
    id                   UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    creation_time        TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    domain               TEXT NOT NULL,
    ip_address           INET NOT NULL,
    requester_ip_address INET NOT NULL,
    requester_contact    TEXT NOT NULL
)
