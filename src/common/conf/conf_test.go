package conf

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"
)

func Test_config(t *testing.T) {

	pf, err := os.Open("server.conf")
	if err != nil {
		return
	}
	defer pf.Close()

	gConf := ServerConf{}
	decoder := json.NewDecoder(pf)
	if err = decoder.Decode(&gConf); err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(gConf)

	return
}
