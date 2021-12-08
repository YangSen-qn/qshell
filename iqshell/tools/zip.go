package tools

import (
	"github.com/qiniu/qshell/v2/iqshell/common/alert"
	"github.com/qiniu/qshell/v2/iqshell/common/log"
	"github.com/qiniu/qshell/v2/iqshell/common/utils"
	"os"
)

type ZipInfo struct {
	ZipFilePath string
	UnzipPath   string
}

// 解压使用mkzip压缩的文件
func Unzip(info ZipInfo) {
	if len(info.ZipFilePath) == 0 {
		log.Error(alert.CannotEmpty("unzip file path", ""))
		return
	}

	var err error
	if len(info.UnzipPath) == 0 {
		info.UnzipPath, err = os.Getwd()
		if err != nil {
			log.Error("Get current work directory failed due to error", err)
			return
		}
	} else {
		if _, statErr := os.Stat(info.UnzipPath); statErr != nil {
			log.Error("Specified <UnzipToDir> is not a valid directory")
			return
		}
	}

	unzipErr := utils.Unzip(info.ZipFilePath, info.UnzipPath)
	if unzipErr != nil {
		log.Error("Unzip file failed due to error", unzipErr)
	}
}