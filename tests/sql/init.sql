CREATE DATABASE IF NOT EXISTS dev;
USE dev;
CREATE TABLE IF NOT EXISTS jobs (
    id INT AUTO_INCREMENT PRIMARY KEY,
    run_id VARCHAR(255)  DEFAULT 'run_id',
    extra_run_id VARCHAR(255)  DEFAULT 'extra_run_id',
    ttr INT DEFAULT 30,
    status VARCHAR(255)  DEFAULT 'PENDING',
    cmd VARCHAR(255)  DEFAULT 'sleep 100',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    )  ENGINE=INNODB;
/*
generating regular jobs
for i in range(3,100):
    print(f"INSERT INTO jobs (ttr) VALUES({i});")

 */
/* Example:
INSERT INTO jobs (ttr, cmd) VALUES(100, 'echo zsdadadasdads;sleep 10');
INSERT INTO jobs (ttr, cmd) VALUES(200, 'for i in {1..1000};do echo $i;done');
INSERT INTO jobs (ttr, cmd) VALUES(300, 'echo 12312 >&2;sleep 1000');
* /

/*
 generating jobs for cancellation

 for i in range(101,201):
    print(f"INSERT INTO jobs (ttr, cmd) VALUES(100,'sleep 100');")
*/
