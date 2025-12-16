package conf

import (
	"encoding/json"
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
	"testing"
)

func Test_config(t *testing.T) {

	pf, err := os.Open("./server.yaml")
	if err != nil {
		return
	}
	defer pf.Close()

	gConf := ServerConf{}
	decoder := yaml.NewDecoder(pf)
	if err = decoder.Decode(&gConf); err != nil {
		fmt.Println(err)
		return
	}

	str, err := json.Marshal(gConf)
	fmt.Println(string(str))

	return
}
