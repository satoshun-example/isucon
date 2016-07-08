# isucon4 qualifier

## first

```shell
# machine
lscpu
free -h
cat /proc/version

# middleware
nginx -V
mysql --version
```


## MySQL slow query

```shell
show global variables like '%slow%';

mysqldumpslow -s t /var/lib/mysql/mysqld-slow.log | less
```

## sysctl

/etc/sysctl.conf

```shell
sysctl --system

# reload
sysctl -p
```


## ulimit

/etc/pam.d/su

```
session    required   pam_limits.so
```

/etc/security/limits.conf

```
nginx       soft    nofile  10000
nginx       hard    nofile  30000

www-data     soft    nofile  10000
www-data     hard    nofile  30000

isucon       soft    nofile  10000
isucon       hard    nofile  30000

root       soft    nofile  10000
root       hard    nofile  30000
```


## nginx

check

```
nginx -V
systemctl status nginx.service
```


## settings

check

```
show variables like '%key_buffer_size%';
show variables like '%innodb_buffer_pool_size%';
```


## score

環境: Vagrant: CPU 4 + memory 4096

- 最初: 300くらい
- Unix domainにする: あまり変わらない        // ボトルネックがMySQLのslow queryなので
- Nginx, MySQL conf更新: あまり変わらない    // ボトルネックがMySQLのslow queryなので
- indexとか張る: 3370くらい                 // ボトルネックがGoのCPU boundになる
- getUserしていた無駄クエリ削除: 4800くらい  // ボトルネックがGoのCPU boundのまま
- martini取り除く: 10515くらい              // CPU, Memoryともに落ち着く
- /indexをnginxのみで捌くように変更: 11000くらい     // CPU, Memoryともに落ち着く
- 遅いqueryをgoの変数でcache(再起動したら壊れるからレギュ違反) // 遅くなる
  - 多分, mutex周りの実装がミスって遅くなった
- 2つ遅いqueryをredisで置換 13000くらい
