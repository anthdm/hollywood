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
profiles. For example, to analyze the CPU profile, we can run:

```