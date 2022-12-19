.PHONY: run-server
run-server:
	docker run --name mysql -e MYSQL_ROOT_PASSWORD=root -p 3306:3306 -d mysql:5.7.39

.PHONY: init-database
init-database:
	mysql -h127.0.0.1 -uroot -proot -e "CREATE DATABASE IF NOT EXISTS j2gg0s; USE j2gg0s;CREATE TABLE IF NOT EXISTS user(id BIGINT NOT NULL, name VARCHAR(125) NOT NULL, age INT NOT NULL DEFAULT 0, sex INT NOT NULL DEFAULT 0, INDEX(name, sex), PRIMARY KEY(id));INSERT INTO user(id, name, age) VALUES (1, 'foo', 27), (2, 'bar', 28), (3, 'baz', 29) ON DUPLICATE KEY UPDATE age=VALUES(age);"

.PHONY: connect
connect:
	mysql -h127.0.0.1 -uroot -proot -Dj2gg0s
