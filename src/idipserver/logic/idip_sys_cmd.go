package logic

import (
	"encoding/json"
	"fmt"
	"github.com/yunjoy-tech/aniwar/src/common/conf"
	"github.com/yunjoy-tech/aniwar/src/common/db"
	"github.com/yunjoy-tech/musae/logger"
)

const (
	GM_GLOBAL_CONFIG_SET     = "global.configset"     // 更新配置
	GM_GLOBAL_CONFIG_GET     = "global.configget"     // 获取配置
	GM_GLOBAL_CONFIG_MAP     = "global.configmap"     // 获取所有配置
	GM_GLOBAL_USERACTOR_SIZE = "global.useractorsize" // 获取useractor数量
	GM_GLOBAL_SVC_SIZE       = "global.svcsize"       // 获取服务进程数量
	GM_GLOBAL_DIRTY_WORD     = "global.dirtyword"     // 脏字文本
	GM_GLOBAL_DEPRECATED_MSG = "global.deprecatedmsg" // 新增临时关闭协议
	GM_GLOBAL_CLOSE_FUNC     = "global.closefunc"     // 新增临时关闭功能
)

var GlobalCmdList = []*GmHelpRsp{
	{GM_GLOBAL_CONFIG_SET, "更新配置中心key", "global.configset <key> <value>"},
	{GM_GLOBAL_CONFIG_GET, "获取配置中心key", "global.configget <key>"},
	{GM_GLOBAL_CONFIG_MAP, "获取所有配置", "global.configmap"},
	{GM_GLOBAL_USERACTOR_SIZE, "获取useractor数量", "global.useractorsize"},
	{GM_GLOBAL_SVC_SIZE, "获取服务进程数量", "global.svcsize"},
	{GM_GLOBAL_DIRTY_WORD, "动态屏蔽词配置", "global.dirtyword <word|word|...>"},
	{GM_GLOBAL_DEPRECATED_MSG, "新增临时关闭协议", "global.deprecatedmsg <msgId|msgId|...>"},
	{GM_GLOBAL_CLOSE_FUNC, "新增临时关闭功能", "global.closefunc <funcId|funcId|...>"},
}

var GlobalCmds = make(map[string]GlobalCmdLogicFunc)

type GlobalCmdLogicFunc = func([]string) (string, error)

func RegisterCmdHandler(name string, handler GlobalCmdLogicFunc) {
	if _, ok := GlobalCmds[name]; !ok {
		GlobalCmds[name] = handler
		logger.Debugf("register cmd: %s", name)
	} else if ok {
		logger.Errorf("Duplicate cmd are registered: %s", name)
	}
}

func (s *IDIPServer) InitCmdHandler() {
	RegisterCmdHandler(GM_GLOBAL_CONFIG_SET, s.handleConfigSet)
	RegisterCmdHandler(GM_GLOBAL_CONFIG_GET, s.handleConfigGet)
	RegisterCmdHandler(GM_GLOBAL_CONFIG_MAP, s.handleConfigMap)
	RegisterCmdHandler(GM_GLOBAL_USERACTOR_SIZE, nil)
	RegisterCmdHandler(GM_GLOBAL_SVC_SIZE, nil)
	RegisterCmdHandler(GM_GLOBAL_DIRTY_WORD, s.handleConfigWordSet)
	RegisterCmdHandler(GM_GLOBAL_DEPRECATED_MSG, s.handleConfigDeprecatedMsgAdd)
	RegisterCmdHandler(GM_GLOBAL_CLOSE_FUNC, s.handleConfigCloseFuncAdd)
}

// 处理指定全服命令逻辑
func (s *IDIPServer) HandleGlobalCmd(cmd string, params []string) (string, error) {
	logger.Debugf("handle global cmd, name: %s, params: %v", cmd, params)
	logicFunc, ok := GlobalCmds[cmd]
	if !ok {
		return "", fmt.Errorf("global cmd not found: %s", cmd)
	}

	return logicFunc(params)
}

func (s *IDIPServer) handleConfigSet(params []string) (string, error) {
	if len(params) < 2 {
		return "params error", fmt.Errorf("params error")
	}
	err := s.SaveToConfigCenter(params[0], params[1])
	if err != nil {
		return err.Error(), err
	}
	return "success", nil
}

// 获取配置
func (s *IDIPServer) handleConfigGet(params []string) (string, error) {
	if len(params) < 1 {
		return "params error", fmt.Errorf("params error")
	}

	val, err := s.GetFromConfigCenter(params[0])
	if err != nil {
		return err.Error(), err
	}

	return val, nil
}

// 将脏字文本存入redis
func (s *IDIPServer) handleConfigWordSet(params []string) (string, error) {
	if len(params) < 2 {
		return "params error", fmt.Errorf("params error")
	}

	var temp = params[1]
	if str, err := s.GetConfigKeyForStr(db.KeyCfgGlobalDirtyWord); err == nil && str != "" {
		temp = str + "|" + temp
	}

	err := s.SaveToConfigCenter(db.KeyCfgGlobalDirtyWord, temp)
	if err != nil {
		return err.Error(), err
	}
	return "success", nil
}

// 新增临时关闭协议
func (s *IDIPServer) handleConfigDeprecatedMsgAdd(params []string) (string, error) {
	if len(params) < 1 {
		return "params error", fmt.Errorf("params error")
	}

	var temp = params[0]
	if str, err := s.GetConfigKeyForStr(db.KeyCfgGlobalDeprecatedMsg); err == nil && str != "" {
		temp = str + "|" + temp
	}

	err := s.SaveToConfigCenter(db.KeyCfgGlobalDeprecatedMsg, temp)
	if err != nil {
		return err.Error(), err
	}

	return "success", nil
}

// 新增临时关闭功能
func (s *IDIPServer) handleConfigCloseFuncAdd(params []string) (string, error) {
	if len(params) < 1 {
		return "params error", fmt.Errorf("params error")
	}

	var temp = params[0]
	if str, err := s.GetConfigKeyForStr(db.KeyCfgGlobalCloseFunc); err == nil && str != "" {
		temp = str + "|" + temp
	}

	err := s.SaveToConfigCenter(db.KeyCfgGlobalCloseFunc, temp)
	if err != nil {
		return err.Error(), err
	}

	return "success", nil
}

func (s *IDIPServer) handleConfigMap(params []string) (string, error) {
	// 获取所有的配置key
	keys := conf.Base().CfgKeys

	ret := make(map[string]string)
	// 查找配置值
	for _, key := range keys {
		val, err := s.GetFromConfigCenter(key)
		if err != nil {
			ret[key] = err.Error()
		} else {
			ret[key] = val
		}
	}

	// 返回数据
	d, err := json.Marshal(ret)
	if err != nil {
		return err.Error(), err
	}
	return string(d), nil
}
