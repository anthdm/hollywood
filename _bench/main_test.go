package main

import (
	"testing"
)

func BenchmarkHollywood(b *testing.B) {
	err := benchmark()
	if err != nil {
		b.Fatal(err)
	}
}

/*
func Benchmark_Latency(b *testing.B) {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError})))
	r := remote.New(remote.Config{ListenAddr: "localhost:2013"})
	e, err := actor.NewEngine(actor.EngineOptRemote(r))
	defer r.Stop()
	if err != nil {
		b.Fatal(err)
	}
	a := e.Spawn(newActor, "actor")
	time.Sleep(10 * time.Millisecond)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		res, err := e.Request(a, &Ping{}, 1*time.Millisecond).Result()
		if err != nil {
			b.Fatal(err)
		}
		if _, ok := res.(*Pong); !ok {
			b.Fatal("unexpected response")
		}
	}
}
*/
