package sdkconstant

const (
	pc_channel = "pc"
	pc_Game_id = 1
	pc_appId   = 1
)

func GenPCUid(uid string) string {
	//return fmt.Sprintf("%s_%d_%s", pc_channel, pc_appId, uid)
	return uid
}

func GenPCChannel() string {
	//return fmt.Sprintf("%s_%d", pc_channel, pc_appId)
	return pc_channel
}
