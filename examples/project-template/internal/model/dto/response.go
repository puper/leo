package dto

import (
	"errors"
	"net/http"

	"server/internal/derrors"
	"server/internal/logging"

	"github.com/gin-gonic/gin"
)

func NewResp(ctx *gin.Context) *Resp {
	return &Resp{
		ctx: ctx,
	}
}

type Resp struct {
	ctx     *gin.Context `json:"-"`
	Code    int          `json:"code"`
	Message string       `json:"message"`
	Data    any          `json:"data"`
	Details any          `json:"details,omitempty"`
}

func (me *Resp) Ok(data any) *Resp {
	me.Code = http.StatusOK * 1000
	me.Message = "ok"
	me.Data = data
	me.Send()
	return me
}

func (me *Resp) Error(err error) *Resp {
	path := me.ctx.FullPath()
	if path == "" {
		path = me.ctx.Request.URL.Path
	}
	var derr *derrors.DetailedError
	if errors.As(err, &derr) {
		me.Code = derr.Code
		me.Message = derr.Message
		me.Details = derr.Details
		logging.GetLogger().With(
			"method", me.ctx.Request.Method,
			"path", path,
			"code", derr.Code,
			"message", derr.Message,
			"err", derr.Error(),
		).Error("request failed")
	} else {
		me.Code = http.StatusInternalServerError * 1000
		me.Message = "系统内部错误"
		me.Details = nil
		logging.GetLogger().With(
			"method", me.ctx.Request.Method,
			"path", path,
			"code", me.Code,
			"message", me.Message,
			"err", err,
		).Error("request failed")
	}
	me.Send()
	return me
}

func (me *Resp) Send() {
	me.ctx.JSON(me.Code/1000, me)
}
