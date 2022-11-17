.PHONY: run-server
run-server:
	docker run --name pg -e POSTGRES_PASSWORD=root -p 5432:5432 -d postgres:15

.PHONY: init-database
init-database:
	PGPASSWORD=root psql -h 127.0.0.1 -U postgres -c "CREATE DATABASE j2gg0s"

.PHONY: init-table
init-table:
	PGPASSWORD=root psql -h 127.0.0.1 -U postgres -d j2gg0s -c "CREATE TABLE visitor(id serial PRIMARY KEY, name VARCHAR(127) NOT NULL, visited_timestamp TIMESTAMP NOT NULL, visited_timestamptz TIMESTAMPTZ NOT NULL);" 

.PHONY: connect
connect:
	PGPASSWORD=root psql -h 127.0.0.1 -U postgres -d j2gg0s

.PHONY: insert
insert:
	PGPASSWORD=root psql -h 127.0.0.1 -U postgres -d j2gg0s -c "INSERT INTO visitor(name, visited_timestamp, visited_timestamptz) VALUES('j2gg0s', '2022-11-17 11:30:00', '2022-11-17 11:30:00');"

.PHONY: insert-tz
insert-tz:
	PGPASSWORD=root psql -h 127.0.0.1 -U postgres -d j2gg0s -c "INSERT INTO visitor(name, visited_timestamp, visited_timestamptz) VALUES('j2gg0s', '2022-11-17 11:33:00+08', '2022-11-17 11:33:00+08');"

.PHONY: select
select:
	PGPASSWORD=root psql -h 127.0.0.1 -U postgres -d j2gg0s -c "SELECT * FROM visitor;"
