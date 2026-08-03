package auth

import (
	"errors"
	"fmt"

	ierr "reservation-sys/internal/errors"
)

var (
	ErrInvalidCode        = errors.New("微信 code 无效")
	ErrInvalidCredentials = errors.New("用户名或密码错误")
	ErrUserNotFound       = fmt.Errorf("%w: 用户不存在", ierr.ErrNotFound)
)
