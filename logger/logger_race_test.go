package logger

import (
	"sync"
	"testing"
)

func TestRaceCloseWithConcurrentLog(t *testing.T) {
	l, err := StartLogger(&LoggerConfig{Type: LogTypeConsole})
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := 0; i < 500; i++ {
			l.Info("concurrent log during shutdown window")
		}
	}()
	go func() {
		defer wg.Done()
		_ = l.Close()
	}()
	wg.Wait()
	_ = l.Close() // idempotent
}
