package testdata

import (
	"testing"
	"time"
)

func TestUsesSleep(t *testing.T) {
	time.Sleep(10 * time.Millisecond)
}
