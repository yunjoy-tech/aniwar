package test

import (
	"fmt"
	"testing"

	"gitlab.musadisca-games.com/wangxw/musae/framework/logger"

	baseconf "gitlab.musadisca-games.com/wangxw/aniwar/src/common/conf"
	excel "gitlab.musadisca-games.com/wangxw/aniwar/src/excel/data"
)

func init() {
	err := baseconf.LoadConf("../../../output/res/server.conf")
	if err != nil {
		panic(err)
	}

	// logger初始化
	if err := logger.Init("log", "test"); err != nil {
		panic(err)
	}

}

func Test0001(t *testing.T) {
	DataDir := "E:\\ss-projects\\go-projects\\aniwar\\output\\res\\data\\"

	err := excel.GetConfigMgr().Load(DataDir)
	if err != nil {
		fmt.Println(err.Error())
	}
}

func Test0002(t *testing.T) {
	DataDir := "E:\\ss-projects\\go-projects\\aniwar\\output\\res\\data"

	err := excel.GetConfigMgr().Load(DataDir)
	if err != nil {
		fmt.Println(err.Error())
	}
	logger.Debugf("测试读取excel， config.ITEM_PACKAGE_LIMIT=%d", excel.GetConfigMgr().GetCfg().ITEM_PACKAGE_LIMIT)

	logger.Debug("after Reload.........................")

	err = excel.GetConfigMgr().Load(DataDir)
	if err != nil {
		fmt.Println(err.Error())
	}
	logger.Debugf("测试读取excel， config.ITEM_PACKAGE_LIMIT=%d", excel.GetConfigMgr().GetCfg().ITEM_PACKAGE_LIMIT)
}
