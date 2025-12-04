package common

type IExcelMgr interface {
	Load(path string) error
	GetFileName() string
}
