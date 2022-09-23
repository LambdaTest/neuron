CREATE TABLE IF NOT EXISTS parsing_queue
(
    id         VARCHAR(32) NOT NULL PRIMARY KEY,
    status     ENUM('ready', 'processing', 'completed') NOT NULL,
    org_id     VARCHAR(32) NOT NULL,
    build_id   VARCHAR(32) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    CONSTRAINT parsing_queue_ibfk_2 UNIQUE (build_id),
    CONSTRAINT parsing_queue_build_id_fk FOREIGN KEY (build_id) REFERENCES
        build (id),
    CONSTRAINT parsing_queue_ibfk_1 FOREIGN KEY (org_id) REFERENCES
        organizations (id)
);

ALTER TABLE build
  ADD payload_address VARCHAR(512) NULL;
