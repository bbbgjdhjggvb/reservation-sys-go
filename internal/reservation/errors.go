package reservation

import (
	"errors"
	"fmt"

	ierr "reservation-sys/internal/errors"
)

var (
	ErrSlotOccupied      = errors.New("时段已被占用")
	ErrOrderNotFound     = fmt.Errorf("%w: 预约不存在", ierr.ErrNotFound)
	ErrInvalidStatus     = errors.New("订单状态不允许此操作")
	ErrDailyLimitReached = errors.New("超过每日预约上限")
)
