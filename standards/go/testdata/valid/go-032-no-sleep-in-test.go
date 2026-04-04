package testdata

import "testing"

func TestNoSleep(t *testing.T) {
	done := make(chan struct{})
	go func() {
		close(done)
	}()
	<-done
}
