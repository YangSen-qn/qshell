package operations

import (
	"github.com/qiniu/qshell/v2/iqshell/common/account"
	"github.com/qiniu/qshell/v2/iqshell/common/alert"
	"github.com/qiniu/qshell/v2/iqshell/common/data"
	"github.com/qiniu/qshell/v2/iqshell/common/log"
	"os"
)

type AddInfo struct {
	Name      string
	AccessKey string
	SecretKey string
	Over      bool
}

func (info *AddInfo) Check() error {
	if len(info.Name) == 0 {
		return alert.CannotEmptyError("Name", "")
	}
	if len(info.AccessKey) == 0 {
		return alert.CannotEmptyError("AccessKey", "")
	}
	if len(info.SecretKey) == 0 {
		return alert.CannotEmptyError("SecretKey", "")
	}
	return nil
}

// 保存账户信息到账户文件中， 并保存在本地数据库
func Add(info AddInfo) {
	acc := account.Account{
		Name:      info.Name,
		AccessKey: info.AccessKey,
		SecretKey: info.SecretKey,
	}

	if err := account.SetAccountToLocalJson(acc); err != nil {
		log.ErrorF("user add: set current error:%v", err)
		os.Exit(data.StatusError)
		return
	}

	if err := account.SaveToDB(acc, info.Over); err != nil {
		log.ErrorF("user add: save user to db error:%v", err)
		os.Exit(data.StatusError)
		return
	}
}
