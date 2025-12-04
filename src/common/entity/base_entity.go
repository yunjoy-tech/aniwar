package entity

// DB对象类的接口
type IBaseEntity interface {
	GetID() uint64
}

// DB对象类的基类
type BaseEntity struct {
	ID uint64 // 表的主键

	CreateSec int64 // 创建时间
	UpdateSec int64 // 更新时间
}
