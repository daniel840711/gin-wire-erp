package path

import (
	"os"
	"path/filepath"
	"runtime"
)

// RootPath 獲取根目錄路徑
// RootPath 傳回專案根目錄的絕對路徑
func RootPath() string {
	// 透過 runtime.Caller(0) 回推到此檔案，再往上三層：/project/utils/path/path.go → /project
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		panic("❌ 無法取得 caller 位置")
	}

	// 返回專案根目錄路徑
	projectRoot := filepath.Clean(filepath.Join(filepath.Dir(filename), "..", ".."))
	return projectRoot
}

// Exists 路径是否存在
func Exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}
