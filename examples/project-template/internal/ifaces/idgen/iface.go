package idgen

import "github.com/bwmarrin/snowflake"

type GenerateID interface {
	GenerateID() snowflake.ID
}
