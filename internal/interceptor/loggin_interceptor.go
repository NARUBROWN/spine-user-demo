package interceptor

import (
	"crypto/rand"
	"encoding/hex"
	"log"
	"strings"
	"time"
	"unicode"

	"github.com/NARUBROWN/spine/core"
	spinectx "github.com/NARUBROWN/spine/pkg/spine"
)

type LoggingInterceptor struct{}

const (
	requestIDKey = "request_id"
	startedAtKey = "request_started_at"
)

func (i *LoggingInterceptor) PreHandle(ctx spinectx.Ctx, meta core.HandlerMeta) error {
	requestID := requestID(httpRequest(ctx).Header("X-Request-ID"))
	ctx.ExecutionContext().Set(requestIDKey, requestID)
	ctx.ExecutionContext().Set(startedAtKey, time.Now())
	log.Printf(
		"request started request_id=%s method=%s path=%s",
		requestID,
		httpRequest(ctx).Method(),
		httpRequest(ctx).Path(),
	)
	return nil
}

func (i *LoggingInterceptor) PostHandle(ctx spinectx.Ctx, meta core.HandlerMeta) {
	controllerName := "<unresolved>"
	methodName := "<unresolved>"
	if meta.ControllerType != nil {
		controllerName = meta.ControllerType.String()
	}
	if meta.MethodName != "" {
		methodName = meta.MethodName
	}

	log.Printf(
		"handler completed request_id=%s method=%s path=%s handler=%s.%s",
		contextString(ctx, requestIDKey, "unknown"),
		httpRequest(ctx).Method(),
		httpRequest(ctx).Path(),
		controllerName,
		methodName,
	)
}

func (i *LoggingInterceptor) BeforeResponse(
	ctx spinectx.Ctx,
	meta core.HandlerMeta,
	executionErr error,
) error {
	return nil
}

func (i *LoggingInterceptor) AfterCompletion(
	ctx spinectx.Ctx,
	meta core.HandlerMeta,
	err error,
) {
	if err != nil {
		log.Printf(
			"request failed request_id=%s method=%s path=%s duration=%s error=%v",
			contextString(ctx, requestIDKey, "unknown"),
			httpRequest(ctx).Method(),
			httpRequest(ctx).Path(),
			duration(ctx),
			err,
		)
		return
	}
	log.Printf(
		"request completed request_id=%s method=%s path=%s duration=%s",
		contextString(ctx, requestIDKey, "unknown"),
		httpRequest(ctx).Method(),
		httpRequest(ctx).Path(),
		duration(ctx),
	)
}

func requestID(candidate string) string {
	candidate = strings.TrimSpace(candidate)
	if candidate != "" && len(candidate) <= 128 && strings.IndexFunc(candidate, unicode.IsControl) == -1 {
		return candidate
	}
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err == nil {
		return hex.EncodeToString(raw[:])
	}
	return "unavailable"
}

func contextString(ctx spinectx.Ctx, key string, fallback string) string {
	value, ok := ctx.ExecutionContext().Get(key)
	if !ok {
		return fallback
	}
	result, ok := value.(string)
	if !ok {
		return fallback
	}
	return result
}

func duration(ctx spinectx.Ctx) time.Duration {
	value, ok := ctx.ExecutionContext().Get(startedAtKey)
	if !ok {
		return 0
	}
	startedAt, ok := value.(time.Time)
	if !ok {
		return 0
	}
	return time.Since(startedAt).Round(time.Microsecond)
}

func httpRequest(ctx spinectx.Ctx) spinectx.HTTPRequestView {
	request, ok := ctx.HTTPRequestContext()
	if !ok {
		panic("HTTP interceptor requires an HTTP request context")
	}
	return request
}

func (*LoggingInterceptor) Constructor() *LoggingInterceptor { return &LoggingInterceptor{} }
