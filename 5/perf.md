# static

## python

```shell
wrk -t1 -c1 -d5 http://127.0.0.1/css/bootstrap.min.css
Running 5s test @ http://127.0.0.1/css/bootstrap.min.css
  1 threads and 1 connections
  Thread Stats   Avg      Stdev     Max   +/- Stdev
    Latency     8.48ms    2.91ms  37.40ms   89.84%
    Req/Sec   119.92     23.68   160.00     70.00%
  598 requests in 5.01s, 70.02MB read
Requests/sec:    119.46
Transfer/sec:     13.99MB
```


## go

```shell
wrk -t1 -c1 -d5 http://127.0.0.1/css/bootstrap.min.css
Running 5s test @ http://127.0.0.1/css/bootstrap.min.css
  1 threads and 1 connections
  Thread Stats   Avg      Stdev     Max   +/- Stdev
    Latency     2.24ms    2.33ms  42.58ms   97.70%
    Req/Sec   491.70     74.20   590.00     74.00%
  2450 requests in 5.01s, 286.88MB read
Requests/sec:    489.16
Transfer/sec:     57.28MB

wrk -t2 -c10 -d5 http://127.0.0.1/css/bootstrap.min.css
Running 5s test @ http://127.0.0.1/css/bootstrap.min.css
  2 threads and 10 connections
  Thread Stats   Avg      Stdev     Max   +/- Stdev
    Latency    18.10ms    8.83ms  79.72ms   81.31%
    Req/Sec   284.53     69.97   430.00     68.00%
  2845 requests in 5.03s, 333.13MB read
Requests/sec:    565.91
Transfer/sec:     66.26MB
```


## nginx

```shell
wrk -t1 -c1 -d5 http://127.0.0.1/css/bootstrap.min.css
Running 5s test @ http://127.0.0.1/css/bootstrap.min.css
  1 threads and 1 connections
  Thread Stats   Avg      Stdev     Max   +/- Stdev
    Latency     2.08ms    2.08ms  33.71ms   97.63%
    Req/Sec   527.32    117.01   710.00     72.00%
  2624 requests in 5.00s, 307.28MB read
Requests/sec:    524.73
Transfer/sec:     61.45MB

wrk -t2 -c10 -d5 http://127.0.0.1/css/bootstrap.min.css
Running 5s test @ http://127.0.0.1/css/bootstrap.min.css
  2 threads and 10 connections
  Thread Stats   Avg      Stdev     Max   +/- Stdev
    Latency   206.29ms  340.32ms   1.37s    82.32%
    Req/Sec   333.44    141.13   720.00     71.91%
  3036 requests in 5.00s, 355.56MB read
Requests/sec:    606.73
Transfer/sec:     71.06MB
```
