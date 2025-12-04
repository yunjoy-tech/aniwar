package utils

import (
	"fmt"
	"testing"
)

func TestBit1(t *testing.T) {
	var num32 int32 = 1
	var bitPos int32 = 2
	val := GetInt32AtBit(num32, bitPos)
	fmt.Println(fmt.Sprintf("%d数字的bit, 第%d位是:%d", num32, bitPos, val))

	var num64 int32 = 8
	bitPos = 4
	val = GetInt32AtBit(num64, bitPos)
	fmt.Println(fmt.Sprintf("%d数字的bit, 第%d位是:%d", num64, bitPos, val))

	var num int32 = 0
	bitPos = 4
	val = GetInt32AtBit(num, bitPos)
	fmt.Println(fmt.Sprintf("%d数字的bit, 第%d位是:%d", num, bitPos, val))
}
