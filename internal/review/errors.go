package review

import (
	"errors"
	"fmt"

	ierr "reservation-sys/internal/errors"
)

var (
	ErrOrderNotFound    = fmt.Errorf("%w: 订单不存在", ierr.ErrNotFound)
	ErrStatusTransition = errors.New("状态转换非法")
	ErrPasswordTooShort = errors.New("密码长度不足")
	ErrNotL1Approved    = errors.New("一级审核未通过，无法设置密码")
)
