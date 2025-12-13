package gmeta

import (
	"gitee.com/bychannel/aniwar/src/meta"
)

var (
	metaMgr = &MetaMgr{}
)

type MetaMgr struct {
	*meta.Tables
}

func GetMetaMgr() *MetaMgr {
	return metaMgr
}

// 加载所有的策划配表数据
func (m *MetaMgr) LoadAllMeta() error {
	tables, err := meta.NewTables(jsonLoader)
	if err != nil {
		return err
	}

	m.Tables = tables
	return nil
}
