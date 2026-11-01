package interceptor

import (
	"crypto/rand"
	"encoding/hex"
	"log"
	"strings"
	"time"
	"unicode"

	"github.com/NARUBROWN/spine/core"
)

type LoggingInterceptor struct{}

const (
	requestIDKey = "request_id"
	startedAtKey = "request_started_at"
)

func (i *LoggingInterceptor) PreHandle(ctx core.ExecutionContext, meta core.HandlerMeta) error {
	requestID := requestID(ctx.Header("X-Request-ID"))
	ctx.Set(requestIDKey, requestID)
	ctx.Set(startedAtKey, time.Now())
	log.Printf(
		"request started request_id=%s method=%s path=%s",
		requestID,
		ctx.Method(),
		ctx.Path(),
	)
	return nil
}

func (i *LoggingInterceptor) PostHandle(ctx core.ExecutionContext, meta core.HandlerMeta) {
	controllerName := "<unresolved>"
	methodName := "<unresolved>"
	if meta.ControllerType != nil {
		controllerName = meta.ControllerType.String()
	}
	if meta.Method.Name != "" {
		methodName = meta.Method.Name
	}

	log.Printf(
		"handler completed request_id=%s method=%s path=%s handler=%s.%s",
		contextString(ctx, requestIDKey, "unknown"),
		ctx.Method(),
		ctx.Path(),
		controllerName,
		methodName,
	)
}

func (i *LoggingInterceptor) BeforeResponse(
	ctx core.ExecutionContext,
	meta core.HandlerMeta,
	executionErr error,
) error {
	return nil
}

func (i *LoggingInterceptor) AfterCompletion(
	ctx core.ExecutionContext,
	meta core.HandlerMeta,
	err error,
) {
	if err != nil {
		log.Printf(
			"request failed request_id=%s method=%s path=%s duration=%s error=%v",
			contextString(ctx, requestIDKey, "unknown"),
			ctx.Method(),
			ctx.Path(),
			duration(ctx),
			err,
		)
		return
	}
	log.Printf(
		"request completed request_id=%s method=%s path=%s duration=%s",
		contextString(ctx, requestIDKey, "unknown"),
		ctx.Method(),
		ctx.Path(),
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

func contextString(ctx core.ExecutionContext, key string, fallback string) string {
	value, ok := ctx.Get(key)
	if !ok {
		return fallback
	}
	result, ok := value.(string)
	if !ok {
		return fallback
	}
	return result
}

func duration(ctx core.ExecutionContext) time.Duration {
	value, ok := ctx.Get(startedAtKey)
	if !ok {
		return 0
	}
	startedAt, ok := value.(time.Time)
	if !ok {
		return 0
	}
	return time.Since(startedAt).Round(time.Microsecond)
}
