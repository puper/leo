package config

type Config struct {
	Servers map[string]struct {
		Driver          string         `json:"driver"`
		Master          string         `json:"master"`
		Slave           []string       `json:"slave"`
		ConnMaxLifeTime int            `json:"connMaxLifeTime"`
		MaxIdleConns    int            `json:"maxIdleConns"`
		MaxOpenConns    int            `json:"maxOpenConns"`
		Migrate         *MigrateConfig `json:"migrate"`
	} `json:"servers"`
}

type MigrateConfig struct {
	TableName                 string `json:"tableName"`
	IDColumnName              string `json:"idColumnName"`
	IDColumnSize              int    `json:"idColumnSize"`
	UseTransaction            bool   `json:"useTransaction"`
	ValidateUnknownMigrations bool   `json:"validateUnknownMigrations"`
}
