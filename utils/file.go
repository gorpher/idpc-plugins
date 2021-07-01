package utils

import (
	"fmt"
	"github.com/gorpher/gone"
	"github.com/rs/zerolog/log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ScanFile 扫描需要加载的文件 filePath需要加载的文件路径， defaultName默认文件名，envName环境变量名
// 扫描顺序 1.命令行参数 2. 环境变量 3.工作路径  4.当前程序同级目录ddl 5.系统环境变量
func ScanFile(filePath, defaultName, envName string) (string, error) {
	if filePath != "" && FileExist(filePath) {
		return filePath, nil
	}
	filePath = os.Getenv(envName)
	if filePath != "" && FileExist(filePath) {
		return filePath, nil
	}
	pwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	filePath = filepath.Join(filepath.Clean(pwd), defaultName)
	if filePath != "" && FileExist(filePath) {
		return filePath, nil
	}
	location, err := os.Executable()
	if err != nil {
		return "", err
	}
	filePath = filepath.Join(filepath.Dir(location), defaultName)
	if filePath != "" && FileExist(filePath) {
		return filePath, nil
	}
	paths := strings.Split(os.Getenv("PATH"), string(os.PathListSeparator))
	gone.AfterStopFunc(time.Second*5, func(c <-chan struct{}) {
		for i := len(paths) - 1; i >= 0; i-- {
			select {
			case <-c:
			default:
				root := paths[i]
				if root != "" && gone.FileIsDir(root) {
					err = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
						if err != nil {
							return err
						}
						if !info.IsDir() && info.Name() == defaultName {
							filePath = path
						}
						return nil
					})
					if err != nil {
						log.Warn().Err(err).Msg("扫描文件夹失败")
						continue
					}
				}
			}
		}
	})
	if filePath != "" && FileExist(filePath) {
		return filePath, nil
	}
	return "", fmt.Errorf("[%s] file does not exist", defaultName)
}

func FileExist(file string) bool {
	_, err := os.Stat(file)
	if err != nil {
		if os.IsExist(err) {
			return true
		}
		if os.IsNotExist(err) {
			return false
		}
		return false
	}
	return true
}
