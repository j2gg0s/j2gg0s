package main

import (
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/go-sql-driver/mysql"
)

func ExampleSafeUpdate() {
	for _, dsn := range []string{
		"root:root@tcp(127.0.0.1:3306)/j2gg0s",
		"root:root@tcp(127.0.0.1:3306)/j2gg0s?sql_safe_updates=1&sql_select_limit=2",
	} {
		db, err := sql.Open("mysql", dsn)
		if err != nil {
			panic(err)
		}

		var (
			safeUpdates int
			selectLimit string
		)
		{
			rows, err := db.Query("SELECT @@sql_safe_updates, @@sql_select_limit")
			if err != nil {
				panic(err)
			}
			for rows.Next() {
				err := rows.Scan(&safeUpdates, &selectLimit)
				if err != nil {
					panic(err)
				}
				fmt.Printf(
					"\nsql_safe_updates -> %d, sql_select_limit -> %s\n",
					safeUpdates, selectLimit)
			}
		}

		if safeUpdates == 1 {
			{
				row, err := db.Query("SHOW CREATE TABLE user;")
				if err != nil {
					panic(err)
				}
				var name, t string
				for row.Next() {
					err := row.Scan(&name, &t)
					if err != nil {
						panic(err)
					}
					fmt.Printf("\n%s\n", t)
				}
			}

			fmt.Printf("\nUPDATE/DELETE without key_column in WHERE\n")
			for _, q := range []string{
				"UPDATE user SET age = 32;",
				"UPDATE user SET age = 32 WHERE age = 27;",
				"DELETE FROM user;",
			} {
				_, err := db.Exec(q)
				if err != nil {
					fmt.Printf("%s -> %s\n", q, strings.TrimSpace(err.Error()))
				}
			}

			fmt.Printf("\nUPDATE/DELETE with key_column is ok\n")
			for _, q := range []string{
				"UPDATE user SET age = 32 WHERE id < 10;",
				"UPDATE user SET age = 32 WHERE name = 'foo';",
			} {
				_, err := db.Exec(q)
				if err != nil {
					panic(err)
				}
				fmt.Printf("%s -> ok\n", q)
			}

			{
				_, err := db.Exec("INSERT INTO user(id, name, age) VALUES (1, 'foo', 27), (2, 'bar', 28), (3, 'baz', 29) ON DUPLICATE KEY UPDATE age=VALUES(age)")
				if err != nil {
					panic(err)
				}
			}

			fmt.Printf("\nSELECT without LIMIT can return up to sql_select_limit(%s) rows\n", selectLimit)
			for _, q := range []string{
				"SELECT * FROM user;",
				"SELECT * FROM user LIMIT 1000;",
			} {
				rows, err := db.Query(q)
				if err != nil {
					panic(err)
				}
				n := 0
				for rows.Next() {
					n += 1
				}
				fmt.Printf("%s -> %d rows\n", q, n)
			}
		}

		_, err = db.Exec("DELETE FROM user WHERE id < 100")
		if err != nil {
			panic(err)
		}
		db.Close()
	}
	// Output:
	// sql_safe_updates -> 0, sql_select_limit -> 18446744073709551615
	//
	// sql_safe_updates -> 1, sql_select_limit -> 2
	//
	// CREATE TABLE `user` (
	//   `id` bigint(20) NOT NULL,
	//   `name` varchar(125) NOT NULL,
	//   `age` int(11) NOT NULL DEFAULT '0',
	//   `sex` int(11) NOT NULL DEFAULT '0',
	//   PRIMARY KEY (`id`),
	//   KEY `name` (`name`,`sex`)
	// ) ENGINE=InnoDB DEFAULT CHARSET=latin1
	//
	// UPDATE/DELETE without key_column in WHERE
	// UPDATE user SET age = 32; -> Error 1175 (HY000): You are using safe update mode and you tried to update a table without a WHERE that uses a KEY column.
	// UPDATE user SET age = 32 WHERE age = 27; -> Error 1175 (HY000): You are using safe update mode and you tried to update a table without a WHERE that uses a KEY column.
	// DELETE FROM user; -> Error 1175 (HY000): You are using safe update mode and you tried to update a table without a WHERE that uses a KEY column.
	//
	// UPDATE/DELETE with key_column is ok
	// UPDATE user SET age = 32 WHERE id < 10; -> ok
	// UPDATE user SET age = 32 WHERE name = 'foo'; -> ok
	//
	// SELECT without LIMIT can return up to sql_select_limit(2) rows
	// SELECT * FROM user; -> 2 rows
	// SELECT * FROM user LIMIT 1000; -> 3 rows
}
