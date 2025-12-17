package gmeta

import (
	"encoding/json"
	"gitee.com/aniwar2/aniwar/src/common/conf"
	"os"
)

func jsonLoader(file string) ([]map[string]interface{}, error) {
	metaDir := conf.Base().MetaDir
	if bytes, err := os.ReadFile(metaDir + file + ".json"); err != nil {
		return nil, err
	} else {
		jsonData := make([]map[string]interface{}, 0)
		if err = json.Unmarshal(bytes, &jsonData); err != nil {
			return nil, err
		}
		return jsonData, nil
	}
}
