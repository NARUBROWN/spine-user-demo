package interceptor

import (
	"log"

	"github.com/NARUBROWN/spine/core"
)

type LoggingInterceptor struct{}

func (i *LoggingInterceptor) PreHandle(ctx core.ExecutionContext, meta core.HandlerMeta) error {
	log.Printf(
		"[Logging Interceptor][REQ] %s %s",
		ctx.Method(),
		ctx.Path(),
	)
	return nil
}

func (i *LoggingInterceptor) PostHandle(ctx core.ExecutionContext, meta core.HandlerMeta) {
	controllerName := "<unresolved>"
	methodName := "<unresolved>"
	if meta.ControllerType != nil {
		controllerName = meta.ControllerType.Name()
	}
	if meta.Method.Name != "" {
		methodName = meta.Method.Name
	}

	log.Printf(
		"[Logging Interceptor][RES] %s %s -> %s.%s OK",
		ctx.Method(),
		ctx.Path(),
		controllerName,
		methodName,
	)
}

func (i *LoggingInterceptor) AfterCompletion(
	ctx core.ExecutionContext,
	meta core.HandlerMeta,
	err error,
) {
	if err != nil {
		log.Printf(
			"[Logging Interceptor][ERR] %s %s : %v",
			ctx.Method(),
			ctx.Path(),
			err,
		)
	}
}
