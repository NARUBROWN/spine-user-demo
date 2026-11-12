package interceptor

import (
	"crypto/subtle"
	"os"
	"strings"

	"github.com/NARUBROWN/spine/core"
	"github.com/NARUBROWN/spine/pkg/httperr"
	spinectx "github.com/NARUBROWN/spine/pkg/spine"
)

// SecurityTokenInterceptor는 선택적 보안 검증 라우트에서 사용할 예제 인증입니다.
// 실제 서비스에서는 사용자별 인증과 권한 검사가 필요합니다.
type SecurityTokenInterceptor struct {
	Token string
}

func (i *SecurityTokenInterceptor) PreHandle(ctx spinectx.Ctx, _ core.HandlerMeta) error {
	header := httpRequest(ctx).Header("Authorization")
	const prefix = "Bearer "
	if i.Token == "" || !strings.HasPrefix(header, prefix) ||
		subtle.ConstantTimeCompare([]byte(strings.TrimPrefix(header, prefix)), []byte(i.Token)) != 1 {
		return httperr.Unauthorized("유효한 인증 정보가 필요합니다.")
	}
	return nil
}

func (*SecurityTokenInterceptor) PostHandle(spinectx.Ctx, core.HandlerMeta) {}
func (*SecurityTokenInterceptor) BeforeResponse(spinectx.Ctx, core.HandlerMeta, error) error {
	return nil
}
func (*SecurityTokenInterceptor) AfterCompletion(spinectx.Ctx, core.HandlerMeta, error) {}

func (*SecurityTokenInterceptor) Constructor() *SecurityTokenInterceptor {
	return &SecurityTokenInterceptor{Token: os.Getenv("SPINE_DEMO_SECURITY_TOKEN")}
}
