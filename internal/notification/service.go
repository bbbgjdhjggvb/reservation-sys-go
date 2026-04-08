/*
 * NotificationService 依赖于 AuthService
 */
package notification

import (
	"reservation-sys/internal/auth"

	"github.com/silenceper/wechat/v2/officialaccount"
)

type NotificationService struct {
	authService *auth.UserAuthService
}

func NewNotificationService(authService *auth.UserAuthService) *NotificationService {
	return &NotificationService{authService: authService}
}

// HandleSubscribe 处理关注业务流
func (s *NotificationService) HandleSubscribe(oa *officialaccount.OfficialAccount, openid string) error {
	_, err := s.authService.FindOrCreate(openid)
	return err
}

// HandleUnsubscribe 处理取消关注业务流
func (s *NotificationService) HandleUnsubscribe(openid string) error {
	return s.authService.SetStatus(openid, false)
}
