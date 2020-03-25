-- install_id is nullable because not all versions of edgectl will report an install_id when generating a new domain,
-- or SCOUT_DISABLE might prevent us from knowing.

ALTER TABLE aes_domains ADD COLUMN install_id TEXT;
