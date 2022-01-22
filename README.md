# fq
a persistent queue library for Go

## Benchmarks

```
$ go test . -bench=. -run Benchmark
goos: linux
goarch: amd64
pkg: github.com/lyonssp/fq
BenchmarkEnqueue5-8     	390165009	         3.05 ns/op
BenchmarkEnqueue10-8    	394515962	         3.05 ns/op
BenchmarkEnqueue50-8    	393034666	         3.04 ns/op
BenchmarkEnqueue100-8   	387193926	         3.11 ns/op
BenchmarkDequeue5-8     	381045183	         3.05 ns/op
BenchmarkDequeue10-8    	388993578	         3.02 ns/op
BenchmarkDequeue50-8    	388552102	         3.16 ns/op
BenchmarkDequeue100-8   	390862748	         3.06 ns/op
```
