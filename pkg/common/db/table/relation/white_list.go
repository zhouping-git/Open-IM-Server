package relation

import (
	"context"
	"time"
)

type WhiteList struct {
	WhiteListId string    `gorm:"column:white_list_id;primary_key;type:varchar(64)" json:"whiteListId" binding:"required"`
	GroupId     string    `gorm:"column:group_id;type:varchar(64);not null;index:index_white_list_group" json:"groupId" binding:"required"`
	UserId      string    `gorm:"column:user_id;type:varchar(64);not null" json:"userId" binding:"required"`
	CreateUser  string    `gorm:"column:create_user;type:varchar(64)" json:"createUser"`
	CreateTime  time.Time `gorm:"column:create_time" json:"createTime"`
}

func (WhiteList) TableName() string {
	return "white_list"
}

type WhiteListInterface interface {
	GetGroupUsers(ctx context.Context, groupId string) ([]*WhiteList, error)
}
