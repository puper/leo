package dto

import "server/internal/modules/example/model/db"

type GetArg struct {
	Id int64 `json:"id,string"`
}

type GetReply struct {
	Example *db.Example `json:"example"`
}

type CreateArg struct {
	Name string `json:"name"`
}

type CreateReply struct {
	Id int64 `json:"id,string"`
}
