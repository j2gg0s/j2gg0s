# [mysql] MySQL's safe update

在使用和维护 MySQL 的过程中, 难免会遇到某些折翼的天使在 UPDATE/DELETE 时没有带 WHERE,
导致大量数据被非预期的更新或删除.
幸运的时, 我们可以通过 MySQL 的配置项 [sql_safe_updates](https://dev.mysql.com/doc/refman/5.7/en/server-system-variables.html#sysvar_sql_safe_updates)
来简单的规避这个问题.

在开启了 [sql_safe_updates]() 后,

- UPDATE 和 DELETE 语句必须在 WHERE 条件中指定 key_column, 或者提供 LIMIT.
- 如果 SELECT 没有指定 LIMIT, 则最多返回 [sql_select_limit](https://dev.mysql.com/doc/refman/5.7/en/server-system-variables.html#sysvar_sql_select_limit) 行数据.
- 涉及多张表的 SELECT, 如果扫描的行数超过 [max_join_size](https://dev.mysql.com/doc/refman/5.7/en/server-system-variables.html#sysvar_max_join_size) 则会被拒绝执行并返回错误.

需要注意, UPDATE/DELETE 即使有 WHERE, 但如果 WHERE 中的列不时 key_column 的话, 依然会报错.
按我的理解, key_column 是指索引字段, 这个索引并不限制是否是 unique.

具体可以参考 [Using Safe-Updates Mode](https://dev.mysql.com/doc/refman/5.7/en/mysql-tips.html#safe-updates) 和 [examples/mysql-safe](./examples/mysql-safe) 中的例子:
```
sql_safe_updates -> 1, sql_select_limit -> 2

CREATE TABLE `user` (
  `id` bigint(20) NOT NULL,
  `name` varchar(125) NOT NULL,
  `age` int(11) NOT NULL DEFAULT '0',
  PRIMARY KEY (`id`),
  KEY `name` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=latin1

UPDATE/DELETE without key_column in WHERE
UPDATE user SET age = 32; -> Error 1175 (HY000): You are using safe update mode and you tried to update a table without a WHERE that uses a KEY column.
UPDATE user SET age = 32 WHERE age = 27; -> Error 1175 (HY000): You are using safe update mode and you tried to update a table without a WHERE that uses a KEY column.
DELETE FROM user; -> Error 1175 (HY000): You are using safe update mode and you tried to update a table without a WHERE that uses a KEY column.

UPDATE/DELETE with key_column is ok
UPDATE user SET age = 32 WHERE id < 10; -> ok
UPDATE user SET age = 32 WHERE name = 'foo'; -> ok

SELECT without LIMIT can return up to sql_select_limit(2) rows
SELECT * FROM user; -> 2 rows
SELECT * FROM user LIMIT 1000; -> 3 rows
```


MySQL 的绝大多数配置项都允许在两个层面配置:
- SESSION, 针对当前链接, 可以覆盖全局配置.
- GLOBAL, 也就是在 server 设置, 并做为链接的默认配置.

[sql_safe_updates]() 默认不开启, 需要主动设置.
[sql_select_limti]() 的默认值也非常的大(18446744073709551615), 等同不限制.
最好的选择自然是在 server 直接设置 safe update,
但如果有时候需要对已运行的项目增加相关功能时, 让客户端灰度增加相关配置会更安全.

[go-sql-driver/mysql](https://github.com/go-sql-driver/mysql) 允许直接在 DSN 中设置 MySQL 的配置项,
如果需要开启 safe update, 仅需要增加如下两个参数, `root:root@tcp(127.0.0.1:3306)/j2gg0s?sql_safe_updates=1&sql_select_limit=2`.
