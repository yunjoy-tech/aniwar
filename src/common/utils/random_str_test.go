package utils

import (
	"fmt"
	"testing"
)

func TestRandomStr(t *testing.T) {
	fmt.Println(RandomStr(10, true, true, true))
	fmt.Println(RandomStr(10, false, true, true))
	fmt.Println(RandomStr(10, false, false, true))
	fmt.Println(RandomStr(10, false, false, false))

	fmt.Println(RandomStr(30, true, true, true))
	fmt.Println(RandomStr(30, false, true, true))
	fmt.Println(RandomStr(30, false, false, true))
	fmt.Println(RandomStr(30, false, false, false))
}
