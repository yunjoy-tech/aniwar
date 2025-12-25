package common

type ReloadParam struct {
	// reload file type
	Type string `json:"type"` // conf,excel
	// reload files
	Files string `json:"files"` // 多个文件通过|符号分割
}
