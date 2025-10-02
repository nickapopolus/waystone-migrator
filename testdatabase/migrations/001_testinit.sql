CREATE TABLE IF NOT EXISTS test (
    id VARCHAR(200) primary key,
    test_name varchar(200) not null
    );

INSERT INTO test (id, test_name) VALUES ('1', 'First Test');
INSERT INTO test (id, test_name) VALUES ('2', 'Second Test');
INSERT INTO test (id, test_name) VALUES ('3', 'Third Test');


-- +down

DROP TABLE IF EXISTS test;