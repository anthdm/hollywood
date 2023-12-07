# Benchmark suite for Hollywood

This is a benchmark suite for the Hollywood framework. It spins up a number of engines, a whole lot of actors
and then sends messages between them.

## Running the benchmark
```
make bench
```

## Profiling the benchmark

We can use the `pprof` tool to profile the benchmark. First, we need to run the benchmark with profiling enabled:
```
make bench-profile
```

This will run the benchmark and generate a CPU and a memory profile. We can then use the `pprof` tool to analyze the
profiles. 


## Analyzing the profiles

### For CPU profile, basic view
```
go tool pprof cpu.prof
> web
```

### For Memory profile, basic view
```
go tool pprof mem.prof
> web
```

### Fancy web interface
```
go tool pprof -http=:8080 cpu.prof
```
and 
```
go tool pprof -http=:8080 mem.prof
```