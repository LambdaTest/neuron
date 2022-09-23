ALTER TABLE build ADD INDEX idx_build_status_created_at (status, created_at);
