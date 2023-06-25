熟悉核心业务系统的每一条 SQL 并了解其面中的索引和涉及的数据是确保系统稳定的必要前提.

站在使用者的角度, 我们可以把数据库执行 SELECT 的过程简化:
- Plan, 数据库根据 SQL 和之前统计的信息确定最优的执行方案, 其中包括具体应该使用哪个 index.
- Scan Index, 数据库通过扫描索引来确定应该从磁盘加载哪些数据到内存中.
- Scan Data, 数据库从磁盘加载数据.
- Filter/Compute, 数据库在内存中, 根据加载的数据进行过滤(where)或计算(order/group).
- 将最终的结果返回给客户端.

此外需要注意:
- MySQL 的索引结构基本是 B-Tree
- 数据库一般会将频繁使用的索引加载到内存中
- 磁盘 IO 相较于内存的耗时高 n 个数量级

## 查询应该尽量使用 index
MySQL 可以将 index 提前加载到内存中, 扫描索引的过程很少涉及磁盘 IO.
相较于从磁盘记载 data, 性能有多个数量级的提升, 而且不消耗宝贵的磁盘 IO 资源.

但请注意使用 index 并不是我们的目的, 减少从磁盘加载数据才是.

我们来看一条优化前的慢 SQL:
- USE INDEX (PRIMARY), 是为了模拟优化前.
- 整张条表的数据在 200k 左右, 这条 SQL 要加载一半的数据.
```sql
EXPLAIN SELECT * FROM vectors USE INDEX (PRIMARY) WHERE id > 605793579343877039 AND origin_type = 'SPU' ORDER BY id ASC LIMIT 100;
+----+-------------+---------+------------+-------+---------------+---------+---------+------+-------+----------+-------------+
| id | select_type | table   | partitions | type  | possible_keys | key     | key_len | ref  | rows  | filtered | Extra       |
+----+-------------+---------+------------+-------+---------------+---------+---------+------+-------+----------+-------------+
|  1 | SIMPLE      | vectors | NULL       | range | PRIMARY       | PRIMARY | 8       | NULL | 96419 |    10.00 | Using where |
+----+-------------+---------+------------+-------+---------------+---------+---------+------+-------+----------+-------------+
```

大多数情况下, 这条查询应该是没有问题的.
- 期望使用的索引是 id, primary key.
- 排序的字段和索引字段相同, 所以不需要从磁盘上加载额外的数据到内存进行排序.
- 带上了 LIMIT, 规避了 OFFSET, 一般而言仅需要从磁盘加载 LIMIT/n 的数据到内存进行过滤即可. n = COUNT(origin_type = 'SPU')/COUNT(*).

但实际结果非常不理想, 原因是 origin_type 为 SPU 的记录占比较低且分布非常不均匀.
导致数据库为了找到 100 条满足 id > 605793579343877039 且 origin_type = 'SPU' 的记录, 将 96k 的数据从磁盘上加载到了内存.
进而导致磁盘 IO 被打满且 SQL 耗时超过 100s. (实例 IOPS 被限制在 600)

```sql
MySQL [vectors]> SELECT COUNT(*) FROM vectors WHERE origin_type = 'SPU' AND id > 605793579343877039;
+----------+
| COUNT(*) |
+----------+
|        0 |
+----------+
1 row in set (0.04 sec)

MySQL [vectors]> SELECT COUNT(*) FROM vectors WHERE id > 605793579343877039;
+----------+
| COUNT(*) |
+----------+
|   105926 |
+----------+
1 row in set (0.04 sec)
```

解决办法是创建 idx_id_origin_type(id, origin_type) 的联合索引, 让数据库可以完全通过内存中的索引完成过滤, 进而最多个从磁盘加载 100 条数据.
```sql
MySQL [vectors]> SELECT * FROM vectors USE INDEX (idx_id_origin_type) WHERE id > 605793579343877039 AND origin_type = 'SPU' ORDER BY id ASC LIMIT 100;
Empty set (0.02 sec)
```
虽然考虑 B-Tree 的特性, idx_origin_type_id(origin_type, id) 可以让上述的查询扫描更少的 index, 但:
- 实际收益非常低, index 大概率已经被加载到内存中, 少扫一点带来的收益很低.
- origin_type 的区分度看上去就非常低.

## 考虑 order 对 index 的影响
对于带 ORDER BY 的查询, 有两种处理方式:
- 通过索引将满足条件的数据全部加载到内存中, 进行排序后返回.
- 通过索引按顺序加载数据到内存中, 进行过滤, 直到有足够的数据后返回.

数据库会自动根据情况进行优化并选择. 但我们在设计时应该尽量确保排序和过滤的字段都可以利用同一个索引, 减少从磁盘加载数据的次数.
```sql
MySQL [vectors]> EXPLAIN SELECT created_at FROM vectors WHERE id > 607373891085739348 ORDER BY created_at LIMIT 10;
+----+-------------+---------+------------+-------+----------------------------+---------+---------+------+------+----------+-----------------------------+
| id | select_type | table   | partitions | type  | possible_keys              | key     | key_len | ref  | rows | filtered | Extra                       |
+----+-------------+---------+------------+-------+----------------------------+---------+---------+------+------+----------+-----------------------------+
|  1 | SIMPLE      | vectors | NULL       | range | PRIMARY,idx_id_origin_type | PRIMARY | 8       | NULL |    1 |   100.00 | Using where; Using filesort |
+----+-------------+---------+------------+-------+----------------------------+---------+---------+------+------+----------+-----------------------------+
1 row in set, 1 warning (0.00 sec)

MySQL [vectors]> EXPLAIN SELECT created_at FROM vectors WHERE id > 0 ORDER BY created_at LIMIT 10;
+----+-------------+---------+------------+-------+----------------------------+----------------+---------+------+------+----------+--------------------------+
| id | select_type | table   | partitions | type  | possible_keys              | key            | key_len | ref  | rows | filtered | Extra                    |
+----+-------------+---------+------------+-------+----------------------------+----------------+---------+------+------+----------+--------------------------+
|  1 | SIMPLE      | vectors | NULL       | index | PRIMARY,idx_id_origin_type | idx_created_at | 4       | NULL |   20 |    50.00 | Using where; Using index |
+----+-------------+---------+------------+-------+----------------------------+----------------+---------+------+------+----------+--------------------------+
1 row in set, 1 warning (0.00 sec)
```

## 理解 B-Tree
自行 google 下, 这是一切的基础, 如果你都无法理解 MySQL 是怎么使用 index 的, 其他一切都没必要.

## 过多的 index 会急剧影响数据的性能
- 插入/更新数据时的额外成本.
- 干扰数据库优化器对索引的选择, 导致非预期结果.
- 过多指, 任何一条非必须的 index.

## 免责
- 距离我上一次正式使用 MySQL/PostgreSQL 应该在 5 年前
- 我不是 DBA, 也没有看过 MySQL/PostgreSQL 的的代码
- 当代数据库的优化已经非常厉害了, 很多你觉得需要从磁盘加载数据的场景其实并不是每次都需要. vectors 这个 case 中有 TEXT, 大概导致每次都需要从磁盘加载数据.
