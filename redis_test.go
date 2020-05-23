package redis

import (
	"fmt"
	"testing"
)

func TestRedis(t *testing.T) {
	pm := NewManager()
	proxy := pm.FastProxy()
	fmt.Println(proxy)
}
