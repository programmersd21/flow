package sampler

import (
	"context"
	"testing"
	"time"
)

func BenchmarkSamplerRunThroughput(b *testing.B) {
	reader := &counterReader{totals: linearTotals(500_000, b.N+100)}
	s := New(reader, 10*time.Microsecond)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	b.ReportAllocs()
	b.ResetTimer()

	go s.Run(ctx)
	for i := 0; i < b.N; i++ {
		<-s.Out
	}
}
