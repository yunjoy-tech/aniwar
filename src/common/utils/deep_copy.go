package utils

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
)

// --- 性能说明 ---
// 自定义拷贝函数 > json > gob

/**
 *  利用gob进行深拷贝
 */
func DeepCopyByGob(src, dst interface{}) error {
	var buffer bytes.Buffer
	if err := gob.NewEncoder(&buffer).Encode(src); err != nil {
		return err
	}
	return gob.NewDecoder(&buffer).Decode(dst)
}

//
// DeepCopyByJson
//  @Description: 利用json进行深拷贝
//  @param src 源数据，必须&取地址传递
//  @param dst 目标数据，必须&取地址传递
//  @return error
//
func DeepCopyByJson(src, dst interface{}) error {
	if tmp, err := json.Marshal(src); err != nil {
		return err
	} else {
		err = json.Unmarshal(tmp, dst)
		return err
	}
}
