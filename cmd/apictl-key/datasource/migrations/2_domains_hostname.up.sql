-- some domain registration are done with CNAME records, pointing to a hostname instead of an IP.

ALTER TABLE aes_domains ALTER COLUMN ip_address DROP NOT NULL;

ALTER TABLE aes_domains ADD COLUMN hostname TEXT;
