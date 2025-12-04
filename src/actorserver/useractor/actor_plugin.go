package useractor

/*
import (
	"errors"
	"fmt"
	"github.com/dapr/go-sdk/actor"
	"os"
	"plugin"
	"runtime"
)

const (
	KeyNewUserActorMode = "NewUserActorMode"
)

var (
	NewUserActorMode func() actor.Server
)

func newUserActorMode() actor.Server {
	actor := &UserActorMode{}
	return actor
}

func init() {
	if runtime.GOOS == "linux" {
		NewUserActorMode = newUserActorMode
		//LoadUserActorPlugin("./output/plugin/useractor.so")
	} else if runtime.GOOS == "windows" {
		NewUserActorMode = newUserActorMode
	}
}

func LoadUserActorPlugin(pluginFile string) error {
	if _, err := os.Stat(pluginFile); os.IsNotExist(err) {
		fmt.Printf("LoadUserActorPlugin, %s not found\n", pluginFile)
		return err
	}

	plug, err := plugin.Open(pluginFile)
	if err != nil {
		fmt.Printf("LoadUserActorPlugin, Open error:%v\n", err)
		return err
	}

	sym, err := GetPluginFuncByName(plug, KeyNewUserActorMode)
	if err != nil || sym == nil {
		fmt.Printf("LoadUserActorPlugin, GetPluginFuncByName %s error:%v\n", KeyNewUserActorMode, err)
		return err
	}
	NewUserActorMode = sym.(func() actor.Server)
	fmt.Printf("LoadUserActorPlugin, load plugin %s succeed\n", pluginFile)
	return nil
}

func GetPluginFuncByName(p *plugin.Plugin, symbolName string) (plugin.Symbol, error) {
	if p == nil {
		return nil, errors.New("plugin nil")
	}
	sym, err := p.Lookup(symbolName)
	if err != nil {
		return nil, err
	}
	return sym, err
}
*/
