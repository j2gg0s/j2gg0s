.PHONY: run-server
run-server:
	docker run -ti --name mysql -e MYSQL_ROOT_PASSWORD=root -p 3306:3306 -d mysql:8.0

.PHONY: init-database
init-database:
	 mysql -h127.0.0.1 -uroot -proot -e "CREATE DATABASE IF NOT EXISTS j2gg0s; USE j2gg0s; CREATE TABLE IF NOT EXISTS visitor(id BIGINT NOT NULL AUTO_INCREMENT, name VARCHAR(125) NOT NULL, visited_timestamp TIMESTAMP NOT NULL, visited_datetime DATETIME NOT NULL, created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP, PRIMARY KEY(id));"

.PHONY: connect
connect:
	mysql -h127.0.0.1 -uroot -proot -D j2gg0s

.PHONY: insert
insert:
	mysql -h127.0.0.1 -uroot -proot -D j2gg0s -e "INSERT INTO visitor(name, visited_timestamp, visited_datetime) VALUES('j2gg0s', '2022-11-16 22:17:00', '2022-11-16 22:17:00');"

.PHONY: select
select:
	mysql -h127.0.0.1 -uroot -proot -D j2gg0s -e "SELECT * FROM visitor;"

.PHONY: select-as-shanghai
select-as-shanghai:
	mysql -h127.0.0.1 -uroot -proot -D j2gg0s -e "SET @@session.time_zone='+08:00'; SELECT * FROM visitor;"
