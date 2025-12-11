package common

import (
	"fmt"

	"github.com/pkg/errors"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/common/utils"

	"gitlab.musadisca-games.com/wangxw/aniwar/src/proto/pb"
)

var (
	UP_DOWN_IDX         = []int32{7}       // 上、下行的标识下标
	SERVICE_IDX         = []int32{6, 5}    // 微服务的标识下标
	GAME_MODEL_IDX      = []int32{4, 3, 2} // 游戏模块的标识下标
	GAME_MODEL_INCR_IDX = []int32{1, 0}    // 各个游戏模块内的自增标识下标
)

// 获取上行/下行标识
func getUpDownFlag(p pb.Protocols) (error, int32) {
	if err := isProtocols(int32(p)); err != nil {
		return err, 0
	}

	val, _ := utils.GetIntWithIndexes(int32(p), UP_DOWN_IDX)
	return nil, val
}

// 获取微服务标识
func getServiceFlag(p pb.Protocols) (error, int32) {
	if err := isProtocols(int32(p)); err != nil {
		return err, 0
	}

	val, _ := utils.GetIntWithIndexes(int32(p), SERVICE_IDX)
	return nil, val
}

// 获取游戏模块标识
func getGameModelFlag(p pb.Protocols) (error, int32) {
	if err := isProtocols(int32(p)); err != nil {
		return err, 0
	}

	val, _ := utils.GetIntWithIndexes(int32(p), GAME_MODEL_IDX)
	return nil, val
}

// 上行和下行协议互转
func convertUpAndDown(p pb.Protocols, addNumber int32) (error, pb.Protocols) {
	if err := isProtocols(int32(p)); err != nil {
		return errors.New(fmt.Sprintf("NOT EXIST before convert, protocol:%v", p)), pb.Protocols_Protocols_None
	}

	cNum := int32(p) + addNumber
	if err := isProtocols(cNum); err != nil {
		return errors.New(fmt.Sprintf("NOT EXIST after convert, before protocol:%v, after protocol:%v", p, cNum)), pb.Protocols_Protocols_None
	}

	return nil, pb.Protocols(cNum)
}

// 是否是上行或下行协议号
func isProtocols(p int32) error {
	_, ok := pb.Protocols_name[p]

	if !ok {
		return errors.New(fmt.Sprintf("NOT EXIST before convert, protocol:%v", p))
	}

	return nil
}

// Up2Down 上行协议号转为对应的下行协议号
func Up2Down(up pb.Protocols) (error, pb.Protocols) {
	return convertUpAndDown(up, 10000000)
}

// UpNumber2DownNumber 上行协议号转为对应的下行协议号
func UpNumber2DownNumber(upNum int32) (error, int32) {
	if err := isProtocols(upNum); err != nil {
		return errors.New(fmt.Sprintf("upNum is NOT protocols, %d", upNum)), 0
	}

	err, down := Up2Down(pb.Protocols(upNum))
	if err != nil {
		return err, int32(down)
	}

	return err, int32(down)
}

// Down2Up 下行协议号转为对应的上行协议号
func Down2Up(up pb.Protocols) (error, pb.Protocols) {
	return convertUpAndDown(up, 10000000*-1)
}

// DownNumber2UpNumber 上行协议号转为对应的下行协议号
func DownNumber2UpNumber(downNum int32) (error, int32) {
	if err := isProtocols(downNum); err != nil {
		return err, 0
	}
	err, up := Down2Up(pb.Protocols(downNum))
	if err != nil {
		return err, int32(up)
	}

	return err, int32(up)
}

// IsUp 是否是上行协议号
func IsUp(p pb.Protocols) bool {
	err, val := getUpDownFlag(p)
	if err != nil {
		return false
	}
	return val == 1
}

// IsDown 是否是下行协议号
func IsDown(p pb.Protocols) bool {
	err, val := getUpDownFlag(p)
	if err != nil {
		return false
	}
	return val == 2
}

func IsUserActorCmd(p pb.Protocols) bool {
	err, flag := getServiceFlag(p)
	if err != nil {
		return false
	}

	return flag == 1 // 01标识UserActor服务上的协议
}

func IsRoomActorCmd(p pb.Protocols) bool {
	err, flag := getServiceFlag(p)
	if err != nil {
		return false
	}

	return flag == 2 // 02标识RoomActor服务上的协议
}

// IsBC 合法的全服广播
func IsBC(p pb.Protocols) bool {
	err, val := getUpDownFlag(p)
	if err != nil {
		return false
	}
	return val == 4
}
