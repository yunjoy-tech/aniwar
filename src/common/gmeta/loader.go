package gmeta

import (
	"encoding/json"
	"io/ioutil"
)

func loader(file string) ([]map[string]interface{}, error) {
	if bytes, err := ioutil.ReadFile("../GenerateDatas/json/" + file + ".json"); err != nil {
		return nil, err
	} else {
		jsonData := make([]map[string]interface{}, 0)
		if err = json.Unmarshal(bytes, &jsonData); err != nil {
			return nil, err
		}
		return jsonData, nil
	}
}
