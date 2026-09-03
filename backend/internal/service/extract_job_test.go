package service

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"
)

func TestExtractConcurrency_FourRunningTwoQueued(t *testing.T) {
	s := NewContractService(nil, nil)
	s.SetExtractConcurrency(4)
	var running atomic.Int32
	var maxRunning atomic.Int32
	started := make(chan string, 6)
	release := make(chan struct{})
	s.extractFn = func(id string) {
		n := running.Add(1)
		for {
			old := maxRunning.Load()
			if n <= old || maxRunning.CompareAndSwap(old, n) {
				break
			}
		}
		started <- id
		<-release
		running.Add(-1)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s.StartExtractWorker(ctx)

	for i := 0; i < 6; i++ {
		s.enqueueExtract(fmt.Sprintf("id-%d", i))
	}
	for i := 0; i < 4; i++ {
		select {
		case <-started:
		case <-time.After(2 * time.Second):
			t.Fatalf("işçi %d başlamadı", i+1)
		}
	}
	select {
	case <-started:
		t.Fatal("5. iş kuyrukta kalmalıydı")
	case <-time.After(150 * time.Millisecond):
	}
	if maxRunning.Load() != 4 {
		t.Fatalf("eşzamanlı = %d, beklenen 4", maxRunning.Load())
	}
	close(release)
	for i := 0; i < 2; i++ {
		select {
		case <-started:
		case <-time.After(2 * time.Second):
			t.Fatal("kuyruktaki işler çalışmadı")
		}
	}
}
