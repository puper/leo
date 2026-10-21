package db

type Example struct {
	Id   int64  `json:"id,string" gorm:"primaryKey;autoIncrement:false"`
	Name string `json:"name"`
}

func (Example) TableName() string      { return "example" }
func (Example) ConnectionName() string { return "server" }
