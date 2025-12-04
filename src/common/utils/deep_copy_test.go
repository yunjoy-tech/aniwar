package utils

import (
	"fmt"
	"testing"
)

func TestDeepCopyByJson(t *testing.T) {
	m := make(map[int32]interface{})
	m[1] = "name"
	m[2] = 999

	mcopy := make(map[int32]interface{})
	err := DeepCopyByJson(&m, &mcopy)
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println(mcopy)
}
