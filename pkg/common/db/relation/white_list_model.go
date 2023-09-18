package relation

import (
	"context"
	"github.com/OpenIMSDK/Open-IM-Server/pkg/common/db/table/relation"
	"github.com/OpenIMSDK/tools/errs"
	"gorm.io/gorm"
)

func NewWhiteList(db *gorm.DB) relation.WhiteListInterface {
	return &WhiteList{NewMetaDB(db, &relation.WhiteList{})}
}

type WhiteList struct {
	*MetaDB
}

func (o WhiteList) GetGroupUsers(ctx context.Context, groupId string) ([]*relation.WhiteList, error) {
	var h []*relation.WhiteList
	return h, errs.Wrap(o.DB.WithContext(ctx).Where("group_id = ?", groupId).Take(&h).Error)
}
