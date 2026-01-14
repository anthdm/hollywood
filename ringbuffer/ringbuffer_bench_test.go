package ringbuffer

import (
	"testing"
	"unsafe"
)

var sinkByte byte

// Payload sizes
type (
	P16  struct{ B [16]byte }
	P256 struct{ B [256]byte }
	P1K  struct{ B [1024]byte }
	P10K struct{ B [10 * 1024]byte }
)

func fillPayload16(i int) P16 {
	var p P16
	p.B[0] = byte(i)
	p.B[15] = byte(i >> 8)
	return p
}
func fillPayload256(i int) P256 {
	var p P256
	p.B[0] = byte(i)
	p.B[255] = byte(i >> 8)
	return p
}
func fillPayload1K(i int) P1K {
	var p P1K
	p.B[0] = byte(i)
	p.B[1023] = byte(i >> 8)
	return p
}
func fillPayload10K(i int) P10K {
	var p P10K
	p.B[0] = byte(i)
	p.B[len(p.B)-1] = byte(i >> 8)
	return p
}

func Benchmark_Compare_PopN_vs_PopNInto_Payload16B(b *testing.B) {
	benchPopNvsPopNIntoPayload(b, int64(1<<20), int64(1024*4), fillPayload16)
}

func Benchmark_Compare_PopN_vs_PopNInto_Payload256B(b *testing.B) {
	benchPopNvsPopNIntoPayload(b, int64(1<<20), int64(1024*4), fillPayload256)
}

func Benchmark_Compare_PopN_vs_PopNInto_Payload1KB(b *testing.B) {
	benchPopNvsPopNIntoPayload(b, int64(1<<20), int64(1024*4), fillPayload1K)
}

func Benchmark_Compare_PopN_vs_PopNInto_Payload10KB(b *testing.B) {
	benchPopNvsPopNIntoPayload(b, int64(1<<20), int64(1024*4), fillPayload10K)
}

// Generic benchmark helper. The "maker" function creates a payload that depends on i
// so the compiler can't constant-fold everything away.
func benchPopNvsPopNIntoPayload[T any](b *testing.B, rbSize, batchSize int64, maker func(int) T) {
	b.Run("PopN", func(b *testing.B) {
		b.ReportAllocs()

		rb := New[T](rbSize)
		for i := int64(0); i < rbSize; i++ {
			rb.Push(maker(int(i)))
		}

		b.ResetTimer()

		var local byte
		for i := 0; i < b.N; i++ {
			msgs, ok := rb.PopN(batchSize)
			if !ok || len(msgs) == 0 {
				b.Fatal("unexpected empty pop")
			}

			// Touch data so it can't be optimized away.
			// We only read a byte-ish worth to keep benchmark focused on queueing.
			for _, v := range msgs {
				local ^= firstByte(&v)
			}

			for _, v := range msgs {
				rb.Push(v)
			}
		}
		sinkByte = local
	})

	b.Run("PopNInto", func(b *testing.B) {
		b.ReportAllocs()

		rb := New[T](rbSize)
		for i := int64(0); i < rbSize; i++ {
			rb.Push(maker(int(i)))
		}

		dst := make([]T, 0, batchSize)

		b.ResetTimer()

		var local byte
		for i := 0; i < b.N; i++ {
			msgs, ok := rb.PopNInto(dst, batchSize)
			if !ok || len(msgs) == 0 {
				b.Fatal("unexpected empty pop")
			}

			for i := range msgs {
				local ^= firstByte(&msgs[i])
			}

			for _, v := range msgs {
				rb.Push(v)
			}

			dst = msgs[:0]
		}
		sinkByte = local
	})
}

// firstByte returns a byte dependent on the value without knowing its shape.
// This keeps the benchmark generic and prevents dead-code elimination.
func firstByte[T any](p *T) byte {
	// This is safe: it reads the first byte of the value's memory representation.
	// We don't interpret it, just use it to keep the compiler honest.
	return *(*byte)(unsafe.Pointer(p))
}
