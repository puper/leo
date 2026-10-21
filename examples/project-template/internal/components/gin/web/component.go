package web

import (
	"context"
	"errors"
	"net"
	"net/http"
	"time"

	"server/internal/components/gin/web/config"

	"github.com/gin-contrib/cors"
	ginzap "github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type Component struct {
	Container
	config    *config.Config
	srv       *http.Server
	listener  net.Listener
	ginEngine *gin.Engine
}

func (me *Component) Close() error {
	if me.srv == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	shutdownErr := me.srv.Shutdown(ctx)
	if shutdownErr != nil {
		me.GetLogger().Desugar().Error("Server forced to shutdown", zap.Error(shutdownErr))
		shutdownErr = errors.Join(shutdownErr, me.srv.Close())
	}
	if me.listener == nil {
		return shutdownErr
	}
	// Close 可能早于异步 Serve 注册监听器，需同步释放组件持有的端口。
	closeErr := me.listener.Close()
	if errors.Is(closeErr, net.ErrClosed) {
		closeErr = nil
	}
	return errors.Join(shutdownErr, closeErr)
}

func MergeConfig(inputCfg *config.Config) *config.Config {
	cfg := &config.Config{}
	if inputCfg != nil {
		*cfg = *inputCfg
	}
	if cfg.Addr == "" {
		cfg.Addr = ":8888"
	}
	if cfg.ReadHeaderTimeout == 0 {
		cfg.ReadHeaderTimeout = 60
	}
	if cfg.ReadTimeout == 0 {
		cfg.ReadTimeout = 120
	}
	if cfg.WriteTimeout == 0 {
		cfg.WriteTimeout = 120
	}
	if cfg.MaxHeaderBytes == 0 {
		cfg.MaxHeaderBytes = 1 << 20
	}
	if cfg.MaxMultipartMemory == 0 {
		cfg.MaxMultipartMemory = 32 << 20
	}
	return cfg
}

func New(container Container, inputCfg *config.Config, configureFuncs ...func(*Component) error) (*Component, error) {
	var err error
	cfg := MergeConfig(inputCfg)
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	me := &Component{
		Container: container,
		config:    cfg,
	}
	me.ginEngine = gin.New()
	me.ginEngine.MaxMultipartMemory = cfg.MaxMultipartMemory

	me.ginEngine.Use(ginzap.Ginzap(me.GetLogger("access").Desugar().WithOptions(zap.WithCaller(false)), time.RFC3339, true))
	me.ginEngine.Use(ginzap.RecoveryWithZap(me.GetLogger().Desugar().WithOptions(zap.WithCaller(false)), true))

	if len(cfg.CorsOrigins) > 0 {
		me.ginEngine.Use(cors.New(cors.Config{
			AllowOrigins:     cfg.CorsOrigins,
			AllowMethods:     []string{"GET", "POST", "OPTIONS"},
			AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
			ExposeHeaders:    []string{"Content-Length"},
			AllowCredentials: true,
			MaxAge:           12 * time.Hour,
		}))
	}

	me.srv = &http.Server{
		Addr:              cfg.Addr,
		Handler:           me.ginEngine,
		ReadHeaderTimeout: time.Duration(cfg.ReadHeaderTimeout) * time.Second,
		ReadTimeout:       time.Duration(cfg.ReadTimeout) * time.Second,
		WriteTimeout:      time.Duration(cfg.WriteTimeout) * time.Second,
		MaxHeaderBytes:    cfg.MaxHeaderBytes,
	}

	for _, configureFunc := range configureFuncs {
		if err := configureFunc(me); err != nil {
			return nil, err
		}
	}
	me.listener, err = net.Listen("tcp", cfg.Addr)
	if err != nil {
		return nil, err
	}
	me.GetLogger().Desugar().Info("Server is ready to handle requests", zap.String("addr", me.listener.Addr().String()))
	go func() {
		if err := me.srv.Serve(me.listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			me.GetLogger().Desugar().Fatal("Server serve failed", zap.Error(err))
		}
	}()
	return me, nil
}

func (me *Component) GetGinEngine() *gin.Engine {
	return me.ginEngine
}
