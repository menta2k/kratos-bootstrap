package bootstrap

import (
	"fmt"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/registry"

	conf "github.com/menta2k/kratos-bootstrap/gen/api/go/conf/v1"
)

// Bootstrap application boot
func Bootstrap(serviceInfo *ServiceInfo) (*conf.Bootstrap, log.Logger, registry.Registrar) {
	// inject command flags
	Flags := NewCommandFlags()
	Flags.Init()

	var err error

	// load configs
	if err = LoadBootstrapConfig(Flags.Conf); err != nil {
		panic(fmt.Sprintf("load config failed: %v", err))
	}

	// init logger
	ll := NewLoggerProvider(commonConfig.Logger, serviceInfo)

	// init registrar
	reg := NewRegistry(commonConfig.Registry)

	// init tracer
	if err = NewTracerProvider(commonConfig.Trace, serviceInfo); err != nil {
		panic(fmt.Sprintf("init tracer failed: %v", err))
	}

	return commonConfig, ll, reg
}
