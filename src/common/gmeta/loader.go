package gmeta

import (
	"encoding/json"
	"io/ioutil"
)

func jsonLoader(file string) ([]map[string]interface{}, error) {
	// TODO 从配置文件中读取
	if bytes, err := ioutil.ReadFile("E:\\aniwar2\\aniwar\\output\\res\\meta\\" + file + ".json"); err != nil {
		return nil, err
	} else {
		jsonData := make([]map[string]interface{}, 0)
		if err = json.Unmarshal(bytes, &jsonData); err != nil {
			return nil, err
		}
		return jsonData, nil
	}
}
