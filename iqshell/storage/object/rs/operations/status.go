package operations

import (
	"fmt"
	"github.com/qiniu/qshell/v2/iqshell/common/export"
	"github.com/qiniu/qshell/v2/iqshell/common/log"
	"github.com/qiniu/qshell/v2/iqshell/common/utils"
	"github.com/qiniu/qshell/v2/iqshell/storage/object/rs"
	"time"
)

type StatusInfo rs.StatusApiInfo

func Status(info StatusInfo) {
	result, err := rs.BatchOne(rs.StatusApiInfo(info))
	if err != nil {
		log.ErrorF("Stat error:%v", err)
		return
	}

	log.Alert(getStatusInfo(info, result))
}

type BatchStatusInfo struct {
	BatchInfo BatchInfo
	Bucket    string
}

func BatchStatus(info BatchStatusInfo) {
	if !prepareToBatch(info.BatchInfo) {
		return
	}

	resultExport, err := export.NewFileExport(export.FileExporterConfig{
		SuccessExportFilePath:  info.BatchInfo.SuccessExportFilePath,
		FailExportFilePath:     info.BatchInfo.FailExportFilePath,
		OverrideExportFilePath: info.BatchInfo.OverrideExportFilePath,
	})
	if err != nil {
		log.ErrorF("get export error:%v", err)
		return
	}

	scanner, err := newBatchScanner(info.BatchInfo)
	if err != nil {
		log.ErrorF("get scanner error:%v", err)
		return
	}

	rs.BatchWithHandler(&batchStatusHandler{
		scanner:      scanner,
		info:         &info,
		resultExport: resultExport,
	})
}

type batchStatusHandler struct {
	scanner      *batchScanner
	info         *BatchStatusInfo
	resultExport *export.FileExporter
}

var _ rs.BatchHandler = (*batchStatusHandler)(nil)

func (b *batchStatusHandler) WorkCount() int {
	return b.info.BatchInfo.Worker
}

func (b *batchStatusHandler) ReadOperation() (rs.BatchOperation, bool) {
	var info rs.BatchOperation = nil

	line, success := b.scanner.scanLine()
	if !success {
		return nil, true
	}

	items := utils.SplitString(line, b.info.BatchInfo.ItemSeparate)
	if len(items) > 0 {
		key := items[0]
		if key != "" {
			info = rs.StatusApiInfo{
				Bucket: b.info.Bucket,
				Key:    key,
			}
		}
	}

	return info, false
}

func (b *batchStatusHandler) HandlerResult(operation rs.BatchOperation, result rs.OperationResult) {
	apiInfo, ok := (operation).(rs.StatusApiInfo)
	if !ok {
		return
	}

	info := StatusInfo(apiInfo)
	if result.Code != 200 || result.Error != "" {
		b.resultExport.Fail().ExportF("%s\t%d\t%s\n", info.Key, result.Code, result.Error)
		log.ErrorF("Status '%s' Failed, Code: %d, Error: %s", info.Key, result.Code, result.Error)
	} else {
		status := fmt.Sprintf("%s\t%d\t%s\t%s\t%d\t%d",
			info.Key, result.FSize, result.Hash, result.MimeType, result.PutTime, result.Type)
		b.resultExport.Success().Export(status)
		log.Alert(status)
	}
}

func (b *batchStatusHandler) HandlerError(err error) {
	log.ErrorF("batch Status error:%v:", err)
}

func getStatusInfo(info StatusInfo, status rs.OperationResult) string {
	statInfo := fmt.Sprintf("%-20s%s\r\n", "Bucket:", info.Bucket)
	statInfo += fmt.Sprintf("%-20s%s\r\n", "Key:", info.Key)
	statInfo += fmt.Sprintf("%-20s%s\r\n", "Hash:", status.Hash)
	statInfo += fmt.Sprintf("%-20s%d -> %s\r\n", "Fsize:", status.FSize, utils.FormatFileSize(status.FSize))

	putTime := time.Unix(0, status.PutTime*100)
	statInfo += fmt.Sprintf("%-20s%d -> %s\r\n", "PutTime:", status.PutTime, putTime.String())
	statInfo += fmt.Sprintf("%-20s%s\r\n", "MimeType:", status.MimeType)
	if status.Type == 0 {
		statInfo += fmt.Sprintf("%-20s%d -> 标准存储\r\n", "FileType:", status.Type)
	} else {
		statInfo += fmt.Sprintf("%-20s%d -> 低频存储\r\n", "FileType:", status.Type)
	}
	return statInfo
}