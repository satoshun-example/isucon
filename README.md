# ISUCON practice

## memo

Use java8

```
sudo update-alternatives --config java
sudo update-alternatives --config javac
```


## benchmark

wrk

```shell
wrk -t2 -c100 -d5 http://127.0.0.1:8081
wrk -t2 -c100 -d5 -H "Accept-Encoding: gzip, deflate" http://127.0.0.1:8081
```

alp

```shell
alp -f /var/log/access.log
```

## go

```
EnvironmentFile=/home/isucon/env.sh

GOMAXPROCS=4 /home/isucon/webapp/go/app
```


## python

```
pip install meinheld
gunicorn -b 127.0.0.1:8080 -w 4 --worker-class="egg:meinheld#gunicorn_worker" app:application
gunicorn -b "unix:/tmp/app_isucon.sock" -w 4 --worker-class="egg:meinheld#gunicorn_worker" app:app
```


## nginx

```nginx
worker_processes  1;

events {
  worker_connections  1024;
}

http {
  upstream app {
    # server 127.0.0.1:8080;
    server unix:/tmp/app_isucon.sock;
  }

  server {
    location / {
      proxy_set_header Host $host;
      proxy_pass http://app;
    }
  }
}
```


## mysql

```
mysqldumpslow -s t /var/log/mysql/mysql.slow
myprofiler -user=root -interval=5
```

ORに対しては, 個別にindexを張る

```sql
mysql> alter table relations add index idx_one(one);
Query OK, 0 rows affected (0.89 sec)
Records: 0  Duplicates: 0  Warnings: 0

mysql> alter table relations add index idx_another(another);
Query OK, 0 rows affected (0.90 sec)
Records: 0  Duplicates: 0  Warnings: 0

mysql> explain SELECT * FROM relations WHERE one = 100 OR another = 200;
+----+-------------+-----------+-------------+--------------------------------+---------------------+---------+------+------+-----------------------------------------------+
| id | select_type | table     | type        | possible_keys                  | key                 | key_len | ref  | rows | Extra                                         |
+----+-------------+-----------+-------------+--------------------------------+---------------------+---------+------+------+-----------------------------------------------+
|  1 | SIMPLE      | relations | index_merge | friendship,idx_one,idx_another | idx_one,idx_another | 4,4     | NULL |  178 | Using union(idx_one,idx_another); Using where |
+----+-------------+-----------+-------------+--------------------------------+---------------------+---------+------+------+-----------------------------------------------+
1 row in set (0.00 sec)
```

```sql
mysql> alter table footprints add index idx_user_id_owner_id_created_at(user_id,owner_id,created_at);
Query OK, 0 rows affected (1.27 sec)
Records: 0  Duplicates: 0  Warnings: 0

mysql> explain SELECT user_id, owner_id, DATE(created_at) AS date, MAX(created_at) AS updated FROM footprints WHERE user_id = 100 GROUP BY user_id, owner_id, DATE(created_at) ORDER BY updated DESC LIMIT 1000;
+----+-------------+------------+------+---------------------------------+---------------------------------+---------+-------+------+-----------------------------------------------------------+
| id | select_type | table      | type | possible_keys                   | key                             | key_len | ref   | rows | Extra                                                     |
+----+-------------+------------+------+---------------------------------+---------------------------------+---------+-------+------+-----------------------------------------------------------+
|  1 | SIMPLE      | footprints | ref  | idx_user_id_owner_id_created_at | idx_user_id_owner_id_created_at | 4       | const |  100 | Using where; Using index; Using temporary; Using filesort |
+----+-------------+------------+------+---------------------------------+---------------------------------+---------+-------+------+-----------------------------------------------------------+
1 row in set (0.00 sec)
```

order by, group byのindex

index

```
ALTER TABLE entries ADD INDEX idx_created_at(created_at);
ALTER TABLE entries DROP INDEX idx_created_at;
```

## systemd

```
sudo systemctl daemon-reload
sudo systemctl restartrestart isuxi.python.service
```


## references

-
-
