// @title           Framework Template API
// @version         1.0
// @description     Go + Gin 框架模板，展示标准项目结构和组件使用示例
// @BasePath        /
// @securityDefinitions.apikey Bearer
// @in              header
// @name            Authorization
// @description     使用 Bearer {token}，令牌由接入项目认证流程签发。
package main

import "server/internal/cmd"

func main() { cmd.Execute() }
