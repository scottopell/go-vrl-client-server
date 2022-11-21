package main

import (
	"fmt"
	"time"

	"github.com/dustin/go-humanize"
	"go.uber.org/atomic"
)

type BlackholeRecorder[T string | []byte] struct {
	start      time.Time
	totalBytes atomic.Float64
}

func (tr *BlackholeRecorder[T]) Consume(in T) (int, error) {
	l := len(in)
	tr.recordNumBytes(l)
	return l, nil
}

func (tr *BlackholeRecorder[T]) recordNumBytes(nBytes int) {
	if time.Time.IsZero(tr.start) {
		tr.start = time.Now()
	}
	tr.totalBytes.Add(float64(nBytes))
}

func (tr *BlackholeRecorder[T]) AvgThroughput() string {
	now := time.Now()

	elapsed := now.Sub(tr.start)
	avgBytes := uint64(tr.totalBytes.Load() / elapsed.Seconds())
	return fmt.Sprintf("%s / second", humanize.Bytes(avgBytes))
}
