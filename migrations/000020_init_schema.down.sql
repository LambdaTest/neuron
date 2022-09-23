DROP TABLE IF EXISTS `parsing_queue`;

ALTER TABLE build
  DROP COLUMN payload_address;
