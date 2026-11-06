package idgen

import (
	"testing"

	"github.com/bwmarrin/snowflake"
)

func TestBuilder(t *testing.T) {
	for _, id := range []int64{0, 1, 1023} {
		node, err := Builder(id)()
		if err != nil {
			t.Fatal(err)
		}
		sf := node.(*snowflake.Node)
		first, second := sf.Generate(), sf.Generate()
		if first == second || first.Node() != id {
			t.Fatalf("invalid generated IDs for node %d", id)
		}
	}
	for _, id := range []int64{-1, 1024} {
		if _, err := Builder(id)(); err == nil {
			t.Fatalf("accepted invalid node ID %d", id)
		}
	}
}
