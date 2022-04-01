package operations

import (
	"github.com/qiniu/qshell/v2/iqshell"
	"github.com/qiniu/qshell/v2/iqshell/common/alert"
	"github.com/qiniu/qshell/v2/iqshell/common/data"
	"github.com/qiniu/qshell/v2/iqshell/common/export"
	"github.com/qiniu/qshell/v2/iqshell/common/log"
	"github.com/qiniu/qshell/v2/iqshell/storage/object"
	"github.com/qiniu/qshell/v2/iqshell/storage/object/batch"
)

type CopyInfo object.CopyApiInfo

func (info *CopyInfo) Check() *data.CodeError {
	if len(info.SourceBucket) == 0 {
		return alert.CannotEmptyError("SourceBucket", "")
	}
	if len(info.SourceKey) == 0 {
		return alert.CannotEmptyError("SourceKey", "")
	}
	if len(info.DestBucket) == 0 {
		return alert.CannotEmptyError("DestBucket", "")
	}
	if len(info.DestKey) == 0 {
		info.DestKey = info.SourceKey
		log.WarningF("No set DestKey and set DestKey to SourceKey:%s", info.SourceKey)
	}
	return nil
}

func Copy(cfg *iqshell.Config, info CopyInfo) {
	if shouldContinue := iqshell.CheckAndLoad(cfg, iqshell.CheckAndLoadInfo{
		Checker: &info,
	}); !shouldContinue {
		return
	}

	result, err := object.Copy((*object.CopyApiInfo)(&info))
	if err != nil {
		log.ErrorF("Copy Failed, '%s:%s' => '%s:%s', Error: %v",
			info.SourceBucket, info.SourceKey,
			info.DestBucket, info.DestKey,
			err)
		return
	}

	if len(result.Error) != 0 {
		log.ErrorF("Copy Failed, '%s:%s' => '%s:%s', Code: %d, Error: %s",
			info.SourceBucket, info.SourceKey,
			info.DestBucket, info.DestKey,
			result.Code, result.Error)
		return
	}

	if result.IsSuccess() {
		log.InfoF("Copy Success, [%s:%s] => [%s:%s]",
			info.SourceBucket, info.SourceKey,
			info.DestBucket, info.DestKey)
	}
}

type BatchCopyInfo struct {
	BatchInfo    batch.Info
	SourceBucket string
	DestBucket   string
}

func (info *BatchCopyInfo) Check() *data.CodeError {
	if err := info.BatchInfo.Check(); err != nil {
		return err
	}

	if len(info.SourceBucket) == 0 {
		return alert.CannotEmptyError("SrcBucket", "")
	}

	if len(info.DestBucket) == 0 {
		return alert.CannotEmptyError("DestBucket", "")
	}

	return nil
}

func BatchCopy(cfg *iqshell.Config, info BatchCopyInfo) {
	if shouldContinue := iqshell.CheckAndLoad(cfg, iqshell.CheckAndLoadInfo{
		Checker: &info,
	}); !shouldContinue {
		return
	}

	exporter, err := export.NewFileExport(info.BatchInfo.FileExporterConfig)
	if err != nil {
		log.Error(err)
		return
	}

	batch.NewHandler(info.BatchInfo).ItemsToOperation(func(items []string) (operation batch.Operation, err *data.CodeError) {
		// 如果只有一个参数，源 key 即为目标 key
		srcKey, destKey := items[0], items[0]
		if len(items) > 1 {
			destKey = items[1]
		}
		if srcKey != "" && destKey != "" {
			return &object.CopyApiInfo{
				SourceBucket: info.SourceBucket,
				SourceKey:    srcKey,
				DestBucket:   info.DestBucket,
				DestKey:      destKey,
				Force:        info.BatchInfo.Overwrite,
			}, nil
		} else {
			return nil, alert.Error("", "")
		}
	}).OnResult(func(operation batch.Operation, result *batch.OperationResult) {
		apiInfo, ok := (operation).(*object.CopyApiInfo)
		if !ok {
			return
		}
		in := (*CopyInfo)(apiInfo)
		if result.Code != 200 || result.Error != "" {
			exporter.Fail().ExportF("%s\t%s\t%d\t%s", in.SourceKey, in.DestKey, result.Code, result.Error)
			log.ErrorF("Copy Failed, '%s:%s' => '%s:%s', Code: %d, Error: %s",
				in.SourceBucket, in.SourceKey,
				in.DestBucket, in.DestKey,
				result.Code, result.Error)
		} else {
			exporter.Success().ExportF("%s\t%s", in.SourceKey, in.DestKey)
			log.InfoF("Copy Success, '%s:%s' => '%s:%s'",
				in.SourceBucket, in.SourceKey,
				in.DestBucket, in.DestKey)
		}
	}).OnError(func(err *data.CodeError) {
		log.ErrorF("Batch copy error:%v:", err)
	}).Start()
}
