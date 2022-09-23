ALTER TABLE `repositories`
    ADD COLUMN `collect_coverage` BOOLEAN DEFAULT 0,
    ADD INDEX idx_repositories_collect_coverage (`collect_coverage`);
