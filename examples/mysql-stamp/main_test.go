package main

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

func ExampleStamp() {
	for i, dsn := range []string{
		"root:root@tcp(127.0.0.1:3306)/j2gg0s?parseTime=true",
		"root:root@tcp(127.0.0.1:3306)/j2gg0s?parseTime=true&loc=Asia%2FShanghai",
		"root:root@tcp(127.0.0.1:3306)/j2gg0s?parseTime=true",
	} {
		db, err := sql.Open("mysql", dsn)
		if err != nil {
			panic(err)
		}
		if i == 2 {
			db.Exec("SET @@session.time_zone='+08:00'")
		}

		rows, err := db.Query("SELECT * FROM visitor")
		if err != nil {
			panic(err)
		}
		for rows.Next() {
			var id int64
			var name string
			var visitedTimeStamp, visitedDateTime time.Time
			var createdAt time.Time
			err := rows.Scan(&id, &name, &visitedTimeStamp, &visitedDateTime, &createdAt)
			if err != nil {
				panic(err)
			}
			fmt.Println(id, name, visitedTimeStamp.Format(time.RFC3339), visitedDateTime.Format(time.RFC3339))
		}
		db.Close()
	}
	// Output:
	// 1 j2gg0s 2022-11-16T22:17:00Z 2022-11-16T22:17:00Z
	// 1 j2gg0s 2022-11-16T22:17:00+08:00 2022-11-16T22:17:00+08:00
	// 1 j2gg0s 2022-11-17T06:17:00Z 2022-11-16T22:17:00Z
}
