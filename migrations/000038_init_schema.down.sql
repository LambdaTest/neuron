ALTER TABLE test 
    MODIFY name TEXT  CHARACTER SET latin1 NOT NULL,
    MODIFY test_locator TEXT  CHARACTER SET latin1 NOT NULL;

ALTER TABLE test_suite MODIFY name TEXT  CHARACTER SET latin1 NOT NULL;