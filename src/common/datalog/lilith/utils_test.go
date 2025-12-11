package lilith

// import (
//	"fmt"
//	"gitlab.musadisca-games.com/wangxw/aniwar/src/proto/pb"
//	"testing"
// )
//
// func TestConvertStruct2Str(t *testing.T) {
//	s := &pb.ItemReward{
//		ItemId: 999,
//		Num:    666,
//	}
//	fmt.Println("s: ", ConvertStruct2Str(s))
//
//	type test struct {
//		Name string
//		Age  int `json:"age"`
//	}
//
//	s2 := test{Name: "aaa", Age: 7788}
//	fmt.Println("s2: ", ConvertStruct2Str(s2))
//
//	a := 11
//	fmt.Println("int:", ConvertStruct2Str(a))
// }
//
// func TestConvertListStruct2Str(t *testing.T) {
//	s := &pb.ItemReward{
//		ItemId: 999,
//		Num:    666,
//	}
//	arr := make([]any, 0)
//	arr = append(arr, s, s)
//	fmt.Println("arr: ", ConvertListStruct2Str(arr))
// }
//
// func TestConvert2Str(t *testing.T) {
//	arr := []uint32{1, 22, 333, 4444}
//	fmt.Println(ConvertList2Str(arr))
// }
//
// func TestConvertMap2Str(t *testing.T) {
//	m := make(map[int32]uint64)
//	m[1] = 1
//	m[2] = 22
//	m[3] = 333
//
//	fmt.Println(ConvertMap2Str(m))
// }
