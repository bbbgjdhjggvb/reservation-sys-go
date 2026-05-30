# 后端测试代码注释
1. 测试的是 **哪个文件的哪个函数**
2. 该函数的功能是什么
3. 测试场景
``` go
// 测试 repository.go 文件中的 
// func FindOrderByID(id uint) (*ReservationOrder, error)
//
// 函数功能：根据订单的 ID 进行查询
func TestFindOrderByID(t *testing.T) {
	t.Run("订单存在_含预加载时段", func(t *testing.T) {
    //  1. SQL 语句是否是
    //     - SELECT * FROM `reservation_orders` WHERE `reservation_orders`.`id` = ? ORDER BY `reservation_orders`.`id` LIMIT ?
    //     - SELECT * FROM `reservation_slots` WHERE `reservation_slots`.`order_id` = ?
    //  2. 测试 order 对象是否正确加载上 slots[] 时间段切片
	})

	t.Run("订单不存在_返回gorm.ErrRecordNotFound", func(t *testing.T) {
    // 1. 当订单不存在时，是否有返回 gorm.ErrRecordNotFound 错误
	})
}
```