package interceptor

import (
	"crypto/subtle"
	"strings"

	"github.com/NARUBROWN/spine/core"
	"github.com/NARUBROWN/spine/pkg/httperr"
)

// SecurityTokenInterceptor는 선택적 보안 검증 라우트에서 사용할 예제 인증입니다.
// 실제 서비스에서는 사용자별 인증과 권한 검사가 필요합니다.
type SecurityTokenInterceptor struct {
	Token string
}

func (i *SecurityTokenInterceptor) PreHandle(ctx core.ExecutionContext, _ core.HandlerMeta) error {
	header := ctx.Header("Authorization")
	const prefix = "Bearer "
	if i.Token == "" || !strings.HasPrefix(header, prefix) ||
		subtle.ConstantTimeCompare([]byte(strings.TrimPrefix(header, prefix)), []byte(i.Token)) != 1 {
		return httperr.Unauthorized("유효한 인증 정보가 필요합니다.")
	}
	return nil
}

func (*SecurityTokenInterceptor) PostHandle(core.ExecutionContext, core.HandlerMeta) {}
func (*SecurityTokenInterceptor) BeforeResponse(core.ExecutionContext, core.HandlerMeta, error) error {
	return nil
}
func (*SecurityTokenInterceptor) AfterCompletion(core.ExecutionContext, core.HandlerMeta, error) {}
