package idgen

import (
	"github.com/bwmarrin/snowflake"
	"github.com/puper/leo/engine"
)

func Builder(nodeID int64) engine.Builder {
	return func() (any, error) { return snowflake.NewNode(nodeID) }
}
