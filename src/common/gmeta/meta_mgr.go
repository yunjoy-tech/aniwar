package gmeta

import (
	cfg "gitlab.musadisca-games.com/wangxw/aniwar/src/meta"
)

var (
	metaMgr = &MetaMgr{}
)

type MetaMgr struct {
	*cfg.Tables
}

func GetMetaMgr() *MetaMgr {
	return metaMgr
}

// 加载所有的策划配表数据
func (m *MetaMgr) LoadAllMeta() error {
	tables, err := cfg.NewTables(loader)
	if err != nil {
		return err
	}

	m.Tables = tables
	return nil
}
