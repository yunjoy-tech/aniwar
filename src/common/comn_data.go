package common

type ReloadParam struct {
	// reload file type
	Type string `json:"type"` // conf,excel
	// reload files
	Files string `json:"files"` // 多个文件通过|符号分割
}

type UserInfo struct {
	Uid    string `json:"uid"`
	GateId string `json:"gateId"`
}

// Version 返回json数据结构
type Notice struct {
	Title   string `json:"title"`
	Content string `json:"content"`
	Time    string `json:"time"`
	Author  string `json:"author"`
}
