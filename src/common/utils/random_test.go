package utils

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"testing"

	"gitee.com/aniwar2/musae/framework/threading"

	"gitee.com/aniwar2/musae/framework/logger"
	baseconf "gitee.com/bychannel/aniwar/src/common/conf"

	excel "gitee.com/bychannel/aniwar/src/excel/data"
)

func init() {
	err := baseconf.LoadConf("../../../output/res/server.conf")
	if err != nil {
		panic(err)
	}

	// logger初始化
	if err := logger.Init("log", "test"); err != nil {
		panic(err)
	}

	// DataDir := "E:\\ss-projects\\go-projects\\aniwar\\output\\res\\data\\"
	// err = excel.LoadAllExcelData(DataDir)
	// if err != nil {
	// 	fmt.Println(err.Error())
	// }
}

func getContentCfg(poolId int32) []*excel.PoolContentCfg {
	target := make([]*excel.PoolContentCfg, 0)
	excel.GetPoolContentMgr().Foreach(func(cfg *excel.PoolContentCfg) bool {
		if cfg.GetPoolId() == poolId {
			target = append(target, cfg)
		}
		return true
	}, true)

	return target
}

func getContentCfg2(all []*excel.PoolContentCfg, rarity int32, up bool) []*excel.PoolContentCfg {
	target := make([]*excel.PoolContentCfg, 0)
	temp := make(map[int32]int32)
	for _, cfg := range all {
		cardRarity, err := GetCardRarityByItemId(cfg.CardId)
		if err != nil || cardRarity != rarity {
			continue
		}
		f := true
		if up && cfg.Up == 0 {
			f = false
		}
		if f {
			target = append(target, cfg)
			temp[cfg.Id] = cfg.Weight
		}
	}
	fmt.Println(fmt.Sprintf("稀有度目标卡池配置id列表: %v", temp))
	return target
}

// 随机抽取卡牌
func randCard(source []*excel.PoolContentCfg) *excel.PoolContentCfg {
	weightMap := make(map[interface{}]int32)
	for _, cfg := range source {
		weightMap[cfg] = cfg.GetWeight()
	}
	targetCfg := RandomMap(weightMap, true)

	if _, ok := targetCfg.(*excel.PoolContentCfg); !ok {
		fmt.Println(fmt.Sprintf("randCard type assert failed. target:%v, weightMap:%+v", targetCfg, weightMap))
		return nil
	}
	return targetCfg.(*excel.PoolContentCfg)
}

// 随机抽取卡牌
func randCard2(source []*excel.PoolContentCfg) *excel.PoolContentCfg {
	weightMap := make(map[int32]int32)
	for _, cfg := range source {
		// weightMap[cfg] = cfg.GetWeight()
		weightMap[cfg.Id] = cfg.Weight
	}
	targetCfg := Temp_RandomMap2(weightMap, true)

	// if _, ok := targetCfg.(*excel.PoolContentCfg); !ok {
	//	fmt.Println(fmt.Sprintf("randCard type assert failed. target:%v, weightMap:%+v", targetCfg, weightMap))
	//	return nil
	// }
	return excel.GetPoolContentMgr().GetById(targetCfg)
}

// 随机抽取卡牌
func randCard3(source []*excel.PoolContentCfg) *excel.PoolContentCfg {
	weightMap := make(map[int32]int32)
	for _, cfg := range source {
		// weightMap[cfg] = cfg.GetWeight()
		weightMap[cfg.Id] = cfg.Weight
	}
	targetCfg := Temp_RandomMap2(weightMap, false)

	// if _, ok := targetCfg.(*excel.PoolContentCfg); !ok {
	//	fmt.Println(fmt.Sprintf("randCard type assert failed. target:%v, weightMap:%+v", targetCfg, weightMap))
	//	return nil
	// }
	return excel.GetPoolContentMgr().GetById(targetCfg)
}

// 随机抽取卡牌
func randCard4(source []*excel.PoolContentCfg) *excel.PoolContentCfg {
	weightMap := make(map[int32]int32)
	for _, cfg := range source {
		// weightMap[cfg] = cfg.GetWeight()
		weightMap[cfg.Id] = cfg.Weight
	}
	targetCfg := Temp_RandomMap4(weightMap, true)

	// if _, ok := targetCfg.(*excel.PoolContentCfg); !ok {
	//	fmt.Println(fmt.Sprintf("randCard type assert failed. target:%v, weightMap:%+v", targetCfg, weightMap))
	//	return nil
	// }
	return excel.GetPoolContentMgr().GetById(targetCfg)
}

func GetCardRarityByItemId(cardItemId int32) (int32, error) {
	// 道具表
	itemCfg := excel.GetItemMgr().GetById(cardItemId)
	if itemCfg == nil {
		return 0, fmt.Errorf("item config not found")
	}
	return GetCardRarityById(itemCfg.SystemId)
}

func GetCardRarityById(cardId int32) (int32, error) {
	// 角色表
	beastarCfg := excel.GetBeastarMgr().GetById(cardId)
	if beastarCfg == nil {
		return 0, fmt.Errorf("card config not found")
	}
	return beastarCfg.GetRarity(), nil
}

/*func TestRandomMap_(t *testing.T) {
	all := getContentCfg(int32(101))
	cfgs := getContentCfg2(all, 3, false)

	repeates := make(map[int32]int32)
	for j := 0; j < 10000; j++ {
		ret := make(map[int32]int32, 0)

		var last int32
		for i := 0; i < 10; i++ {
			card := randCard(cfgs)
			if card == nil {
				fmt.Println("====空")
				continue
			}

			if last == card.CardId {
				ret[card.CardId] += 1
			} else {
				last = card.CardId
			}
		}

		fmt.Println(fmt.Sprintf("连续重复次数：%v", ret))

		total := int32(0)
		for _, each := range ret {
			total += each
		}
		logger.Infof("总的连续重复次数：%d", total)

		repeates[total] += 1
	}
	logger.Infof("累计连续重复次数：%d", repeates)
	//累计连续重复次数：map[0:3035 1:3871 2:2207 3:725 4:142 5:17 6:3]
	//累计连续重复次数：map[0:2966 1:3922 2:2219 3:719 4:154 5:20]
	//累计连续重复次数：map[0:2904 1:3899 2:2237 3:761 4:168 5:25 6:6]
	//累计连续重复次数：map[0:2972 1:3825 2:2277 3:751 4:149 5:24 6:2]
	//累计连续重复次数：map[0:2964 1:3937 2:2163 3:752 4:154 5:29 6:1]
}

func TestRandomMap_2(t *testing.T) {
	all := getContentCfg(int32(101))
	cfgs := getContentCfg2(all, 3, false)

	repeates := make(map[int32]int32)
	for j := 0; j < 10000; j++ {
		ret := make(map[int32]int32, 0)

		var last int32
		for i := 0; i < 10; i++ {
			card := randCard2(cfgs)
			if card == nil {
				fmt.Println("====空")
				continue
			}

			if last == card.CardId {
				ret[card.CardId] += 1
			} else {
				last = card.CardId
			}
		}

		fmt.Println(fmt.Sprintf("连续重复次数：%v", ret))

		total := int32(0)
		for _, each := range ret {
			total += each
		}
		logger.Infof("总的连续重复次数：%d", total)

		repeates[total] += 1
	}
	logger.Infof("累计连续重复次数：%d", repeates)
	//累计连续重复次数：map[0:3000 1:3844 2:2261 3:711 4:156 5:24 6:4]
	//累计连续重复次数：map[0:2983 1:3933 2:2145 3:758 4:158 5:17 6:6]
	//累计连续重复次数：map[0:2986 1:3797 2:2283 3:746 4:161 5:24 6:3]
	//累计连续重复次数：map[0:3051 1:3879 2:2151 3:732 4:170 5:16 6:1]
	//累计连续重复次数：map[0:3043 1:3916 2:2133 3:726 4:151 5:27 6:4]
}

func TestRandomMap_3(t *testing.T) {
	all := getContentCfg(int32(101))
	cfgs := getContentCfg2(all, 3, false)

	repeates := make(map[int32]int32)
	for j := 0; j < 10000; j++ {
		ret := make(map[int32]int32, 0)

		var last int32
		for i := 0; i < 10; i++ {
			card := randCard3(cfgs)
			if card == nil {
				fmt.Println("====空")
				continue
			}

			if last == card.CardId {
				ret[card.CardId] += 1
			} else {
				last = card.CardId
			}
		}

		fmt.Println(fmt.Sprintf("连续重复次数：%v", ret))

		total := int32(0)
		for _, each := range ret {
			total += each
		}
		//logger.Infof("总的连续重复次数：%d", total)

		repeates[total] += 1
	}
	logger.Infof("累计连续重复次数：%d", repeates)
	//累计连续重复次数：map[0:3084 1:3835 2:2204 3:694 4:161 5:18 6:4]
	//累计连续重复次数：map[0:2926 1:3853 2:2281 3:773 4:142 5:21 6:3 7:1]
	//累计连续重复次数：map[0:2970 1:3915 2:2218 3:734 4:136 5:26 6:1]
}

func TestRandomMap_4(t *testing.T) {
	all := getContentCfg(int32(101))
	cfgs := getContentCfg2(all, 3, false)

	repeates := make(map[int32]int32)
	for j := 0; j < 10000; j++ {
		ret := make(map[int32]int32, 0)

		var last int32
		var eachIdStr string
		for i := 0; i < 10; i++ {
			card := randCard4(cfgs)
			if card == nil {
				fmt.Println("====空")
				continue
			}

			if eachIdStr != "" {
				eachIdStr += ", "
			}
			eachIdStr += strconv.Itoa(int(card.CardId))

			if last == card.CardId {
				if ret[card.CardId] == 0 {
					ret[card.CardId] = 1
				}
				ret[card.CardId] += 1
			} else {
				last = card.CardId
			}
		}

		logger.Infof("each --- ：%v", eachIdStr)
		//fmt.Println(fmt.Sprintf("连续重复次数：%v", ret))

		//total := int32(0)
		//for _, each := range ret {
		//	total += each
		//}
		//logger.Infof("总的连续重复次数：%d", total)
		for _, total := range ret {
			if total == 5 || total == 6 || total == 7 {
				printContinueResult(ret, total)
			}

			repeates[total] += 1
		}
	}
	logger.Infof("累计连续重复次数：%d", repeates)

	// targetCfg := Temp_RandomMap4(weightMap, false)
	//累计连续重复次数：map[0:2908 1:3886 2:2268 3:741 4:179 5:16 6:2]
	//累计连续重复次数：map[0:3014 1:3848 2:2218 3:733 4:154 5:32 6:1]
	//累计连续重复次数：map[0:3005 1:3878 2:2237 3:720 4:128 5:30 6:2]
	//累计连续重复次数：map[0:2990 1:3904 2:2208 3:718 4:160 5:19 6:1]
	//累计连续重复次数：map[0:3031 1:3904 2:2174 3:723 4:147 5:20 6:1]
	//累计连续重复次数：map[0:2944 1:3981 2:2134 3:772 4:148 5:18 6:2 7:1]
	//累计连续重复次数：map[0:3007 1:3872 2:2198 3:727 4:164 5:30 6:2]

	// targetCfg := Temp_RandomMap4(weightMap, true)
	//累计连续重复次数：map[0:3012 1:3870 2:2214 3:752 4:134 5:16 6:2]
	//累计连续重复次数：map[0:3088 1:3786 2:2188 3:730 4:177 5:27 6:4]
	//累计连续重复次数：map[0:3123 1:3819 2:2152 3:719 4:160 5:25 6:2]
	//累计连续重复次数：map[0:2998 1:3914 2:2159 3:762 4:144 5:20 6:3]
}*/

func printContinueResult(ret map[int32]int32, _count int32) {
	strMap := make(map[int32]string)
	for id, count := range ret {
		if strMap[count] != "" {
			strMap[count] += ", "
		}
		strMap[count] += strconv.Itoa(int(id))
	}

	for count, intro := range strMap {
		logger.Infof("连续出现%d次：%s", count, intro)
	}
}

func TestRandomMap(t *testing.T) {
	m := make(map[int32]int32)
	m[101] = 20
	m[102] = 20
	m[103] = 20
	m[104] = 20
	m[105] = 20
	m[106] = 20
	m[107] = 20
	m[108] = 20

	repeates := make(map[int32]int32)

	wg := sync.WaitGroup{}

	for k := 0; k < 10000; k++ {
		wg.Add(1)
		threading.RunSafeWithParam(func(repeates interface{}) {
			_repeates := repeates.(map[int32]int32)

			for j := 0; j < 1000; j++ {
				_temp := make(map[int32]int32, 0)
				ret := make([]string, 0)

				var last int32
				var eachIdStr string
				for i := 0; i < 10; i++ {
					randKey := Temp_RandomMap2(m, true)
					// fmt.Println(randKey)
					// logger.Debugf("%v", randKey)
					// if randKey == nil {
					//	t.Fail()
					// }
					if eachIdStr != "" {
						eachIdStr += ", "
					}
					eachIdStr += strconv.Itoa(int(randKey))

					if last == randKey {
						if _temp[randKey] == 0 {
							_temp[randKey] = 1
						}
						_temp[randKey] += 1
					} else {
						last = randKey

						for key, count := range _temp {
							ret = append(ret, fmt.Sprintf("%d-%d", key, count))
						}
						_temp = make(map[int32]int32, 0)
					}
				}

				logger.Infof("each --- ：%v", eachIdStr)
				// fmt.Println(fmt.Sprintf("连续重复次数：%v", ret))

				// each := int32(0)
				// for _, each := range ret {
				//	each += each
				// }
				// logger.Infof("总的连续重复次数：%d", each)
				for _, each := range ret {
					// if each == 5 || each == 6 || each == 7 {
					// }

					split := strings.Split(each, "-")
					total, _ := strconv.Atoi(split[1])
					logger.Infof("连续出现%d次:%s", total, split[0])

					_repeates[int32(total)] += 1
				}
			}
			wg.Done()
		}, repeates)
	}

	wg.Wait()

	logger.Infof("累计连续重复次数：%d", repeates)
	// 1000*10000次，累计连续重复次数：map[2:7790342 3:854616 4:91961 5:9487 6:1014 7:103 8:8]
	// 1000*10000次，累计连续重复次数：map[2:7794526 3:853875 4:91980 5:9619 6:930 7:88 8:8]
}

func Test22(t *testing.T) {
	all := getContentCfg(int32(101))
	cfgs := getContentCfg2(all, 3, false)
	totalCount := 0

	repeates := make(map[int32]int32)

	for k := 0; k < 1000; k++ {
		for j := 0; j < 100; j++ {
			totalCount++

			_temp := make(map[int32]int32, 0)
			ret := make([]string, 0)

			var last int32
			var eachIdStr string
			for i := 0; i < 10; i++ {
				// randKey := Temp_RandomMap2(m, true)
				card := randCard4(cfgs)
				if card == nil {
					fmt.Println("====空")
					continue
				}
				randKey := card.CardId

				if eachIdStr != "" {
					eachIdStr += ", "
				}
				eachIdStr += strconv.Itoa(int(randKey))

				if last == randKey {
					if _temp[randKey] == 0 {
						_temp[randKey] = 1
					}
					_temp[randKey] += 1
				} else {
					last = randKey

					for key, count := range _temp {
						ret = append(ret, fmt.Sprintf("%d-%d", key, count))
					}
					_temp = make(map[int32]int32, 0)
				}
			}

			logger.Infof("each --- ：%v", eachIdStr)

			for _, each := range ret {
				// if each == 5 || each == 6 || each == 7 {
				// }

				split := strings.Split(each, "-")
				total, _ := strconv.Atoi(split[1])
				logger.Infof("连续出现%d次:%s", total, split[0])

				repeates[int32(total)] += 1
			}
		}
	}

	logger.Infof("10连抽卡:%d, 累计连续重复次数：%d", totalCount, repeates)
	// 10连抽卡:100000, 累计连续重复次数：map[2:77578 3:8564 4:955 5:109 6:11 7:2]
}

func TestRandomList(t *testing.T) {
	source := make([]int32, 0)
	source = append(source, 1, 2, 3, 4, 5, 6)

	result := make([]int32, 0, 30)
	for i := 0; i < 30; i++ {
		result = append(result, RandomList(source))
	}
	fmt.Println(source)
	fmt.Println(result)

	empty := RandomList([]int32{})
	fmt.Println("empty:", empty)
}

func TestRandomListN(t *testing.T) {
	source := make([]int32, 0)
	source = append(source, 1, 2, 3, 4, 5, 6)
	fmt.Println(source)

	fmt.Println(source)
	fmt.Println("random 0:", RandomListN(source, 0))

	fmt.Println(source)
	fmt.Println("random 1:", RandomListN(source, 1))

	fmt.Println(source)
	fmt.Println("random 2:", RandomListN(source, 2))

	fmt.Println(source)
	fmt.Println("random 3:", RandomListN(source, 3))

	fmt.Println(source)
	fmt.Println("random 4:", RandomListN(source, 4))

	fmt.Println(source)
	fmt.Println("random 5:", RandomListN(source, 5))

	fmt.Println(source)
	fmt.Println("random 6:", RandomListN(source, 6))

	fmt.Println(source)
	fmt.Println("random 8:", RandomListN(source, 8))
}

func TestIsSuccessByPercentage(t *testing.T) {
	var rate int32 = 30
	btrue := 0
	bfalse := 0
	for i := 0; i < 10000; i++ {
		if IsSuccessByPercentage(rate) {
			btrue++
		} else {
			bfalse++
		}
	}

	fmt.Println(btrue)
	fmt.Println(bfalse)
}

func Test_RandomByWeights(t *testing.T) {
	weights := make([]int32, 0)
	weights = append(weights, 100, 100, 100, 100)

	valsInt := make([]int, 0)
	valsInt = append(valsInt, 1, 2, 3, 4)
	val, _ := RandomByWeights(valsInt, weights)
	fmt.Printf("valsInt:%+v, ==>> :%d\n", valsInt, val)
}

func Test_RandomInt(t *testing.T) {
	for i := 0; i < 100; i++ {
		randWeight, _ := RandomInt(0, 100)
		fmt.Println(fmt.Sprintf("%d ----%d ", i, uint32(randWeight)))
	}
}
