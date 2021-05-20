### Integration tests for Supraworker
We create an API & supraworker consumes it

### Profile
Links:

* [Profiling Go programs with pprof](https://jvns.ca/blog/2017/09/24/profiling-go-with-pprof/)
* [How I investigated memory leaks in Go using pprof on a large codebase](https://www.freecodecamp.org/news/how-i-investigated-memory-leaks-in-go-using-pprof-on-a-large-codebase-4bec4325e192/)
* http://localhost:8088/debug/pprof/goroutine?debug=1
```shell
go tool pprof -http=:8090  http://localhost:8088/debug/pprof/goroutine

go tool pprof -http=:8090  http://localhost:8088/debug/pprof/heap?nodefraction=0
```
### Check jobs data
```shell
docker exec -ti tests_db_1 mysql -uroot -ptest -D dev -e 'select * from jobs' 
```