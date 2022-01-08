# fq
a persistent queue library for Go

## Benchmarks

```
goos: linux
goarch: amd64
pkg: github.com/lyonssp/fq
BenchmarkEnqueue5-8     	  979857	      1170 ns/op
BenchmarkEnqueue10-8    	 1000000	      1166 ns/op
BenchmarkEnqueue50-8    	  991134	      1209 ns/op
BenchmarkEnqueue100-8   	  964806	      1219 ns/op
BenchmarkDequeue5-8     	 1000000	      1150 ns/op
BenchmarkDequeue10-8    	 1000000	      1167 ns/op
BenchmarkDequeue50-8    	 1000000	      1203 ns/op
BenchmarkDequeue100-8   	  965935	      1221 ns/op
PASS
ok  	github.com/lyonssp/fq	9.498s
```
