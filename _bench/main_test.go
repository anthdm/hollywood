package main

import "testing"

func BenchmarkYourFunction(b *testing.B) {
	err := benchmark()
	if err != nil {
		b.Fatal(err)
	}
}
