package ui

import (
	"testing"
)

func BenchmarkModelViewHero(b *testing.B) {
	m := newTestModel(b, 100, 30)
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = m.View()
	}
}

func BenchmarkModelViewCompact(b *testing.B) {
	m := newTestModel(b, 80, 10)
	m.viewMode = ViewCompact
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = m.View()
	}
}
