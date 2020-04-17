-- edgectl_install_id and aes_install_id are nullable because not all versions of edgectl will report an install_id
-- when generating a new domain, or SCOUT_DISABLE might prevent us from knowing.

ALTER TABLE aes_domains ADD COLUMN edgectl_install_id TEXT;
ALTER TABLE aes_domains ADD COLUMN aes_install_id TEXT;
