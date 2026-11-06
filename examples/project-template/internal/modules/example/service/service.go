package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"unicode/utf8"

	"server/internal/derrors"
	dbiface "server/internal/ifaces/db"
	"server/internal/ifaces/idgen"
	"server/internal/modules/example/model/db"
	"server/internal/modules/example/model/dto"

	"github.com/puper/servicecaller"
	"gorm.io/gorm"
)

type Container interface {
	dbiface.GetDb
	idgen.GenerateID
}

type ExampleService struct{ Container }

func New(container Container) *servicecaller.ServiceCaller {
	sc := servicecaller.New()
	sc.Register("example", &ExampleService{Container: container})
	return sc
}

// Get 查询示例详情。
// @Summary 查询示例详情
// @Description 仅在配置 db、启用 MySQL 示例时注册。
// @Tags example
// @Accept json
// @Produce json
// @Security Bearer
// @Param body body dto.GetArg true "示例 ID"
// @Success 200 {object} dto.Resp{data=dto.GetReply}
// @Failure 400,401,404,500 {object} dto.Resp
// @Router /api/example/detail [post]
func (me *ExampleService) Get(ctx context.Context, args json.RawMessage, reply *dto.GetReply) error {
	var arg dto.GetArg
	if err := json.Unmarshal(args, &arg); err != nil {
		return derrors.NewParamError(err)
	}
	if arg.Id <= 0 {
		return derrors.NewParamError(errors.New("id must be positive"))
	}
	var model db.Example
	err := me.GetDb().ReadModel(&model).WithContext(ctx).First(&model, arg.Id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return derrors.NewNotFoundError(err)
	}
	if err != nil {
		return err
	}
	reply.Example = &model
	return nil
}

// Create 创建示例记录。
// @Summary 创建示例记录
// @Description 仅在配置 db、启用 MySQL 示例时注册。
// @Tags example
// @Accept json
// @Produce json
// @Security Bearer
// @Param body body dto.CreateArg true "示例名称（1–100 字符）"
// @Success 200 {object} dto.Resp{data=dto.CreateReply}
// @Failure 400,401,500 {object} dto.Resp
// @Router /api/example/create [post]
func (me *ExampleService) Create(ctx context.Context, args json.RawMessage, reply *dto.CreateReply) error {
	var arg dto.CreateArg
	if err := json.Unmarshal(args, &arg); err != nil {
		return derrors.NewParamError(err)
	}
	arg.Name = strings.TrimSpace(arg.Name)
	if arg.Name == "" || utf8.RuneCountInString(arg.Name) > 100 {
		return derrors.NewParamError(errors.New("name must contain 1 to 100 characters"))
	}
	model := &db.Example{Id: int64(me.GenerateID()), Name: arg.Name}
	if err := me.GetDb().WriteModel(model).WithContext(ctx).Create(model).Error; err != nil {
		return err
	}
	reply.Id = model.Id
	return nil
}
