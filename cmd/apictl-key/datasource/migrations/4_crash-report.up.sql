CREATE TABLE crash_report (
    id                    UUID PRIMARY KEY NOT NULL,
    creation_time         TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    requester_ip_address  INET NOT NULL,
    metadata              JSONB NOT NULL DEFAULT '{}'::JSONB
)
