ALTER TABLE `repositories`
    DROP INDEX idx_repositories_collect_coverage,
    DROP COLUMN `collect_coverage`;
