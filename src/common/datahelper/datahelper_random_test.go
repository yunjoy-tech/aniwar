package datahelper

import (
	"fmt"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/excel/data"
	"log"
	"testing"
)

func Test_RandomByWeightVo2(t *testing.T) {
	vos2 := make([]*data.WeightVo2, 0)

	for i := 1; i <= 10; i++ {
		vos2 = append(vos2, &data.WeightVo2{
			Weight: int32(i * 10),
			VoId:   int32(i),
			Num:    int32(1000 + i),
		})
	}

	for i := 0; i < 100; i++ {
		vo2, err := RandomByWeightVo2(vos2)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(fmt.Sprintf("%d ----   %+v", i, &vo2))
		//time.Sleep(time.Second)
	}
}
