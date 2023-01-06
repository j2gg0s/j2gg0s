.PHONY: run-server
run-server:
	docker run --name mysql -e MYSQL_ROOT_PASSWORD=root -p 3306:3306 -d mysql:8.0

.PHONY: connect
connect:
	mysql -h127.0.0.1 -uroot -proot -D exp_enum

.PHONY: init-database
init-database:
	mysql -h127.0.0.1 -uroot -proot -e "CREATE DATABASE IF NOT EXISTS exp_enum; USE exp_enum; CREATE TABLE IF NOT EXISTS visitor(name VARCHAR(125) NOT NULL, sex ENUM('MALE', 'FEMALE'), PRIMARY KEY(name));"

.PHONY: insert-normal
insert-normal:
	mysql -h127.0.0.1 -uroot -proot -D j2gg0s -e "USE exp_enum; INSERT INTO visitor(name, sex) VALUES('male', 'MALE'), ('female', 'FEMALE') ON DUPLICATE KEY UPDATE sex=VALUES(sex);"

# error
.PHONY: insert-empty
insert-empty:
	mysql -h127.0.0.1 -uroot -proot -D j2gg0s -e "USE exp_enum; INSERT INTO visitor(name, sex) VALUES('empty', '') ON DUPLICATE KEY UPDATE sex=VALUES(sex);"

.PHONY: insert-null
insert-null:
	mysql -h127.0.0.1 -uroot -proot -D j2gg0s -e "USE exp_enum; INSERT INTO visitor(name, sex) VALUES('null', NULL) ON DUPLICATE KEY UPDATE sex=VALUES(sex);"
