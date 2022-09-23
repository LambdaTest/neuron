ALTER TABLE build ADD COLUMN tag enum('flaky','postmerge', 'premerge') NOT NULL;
