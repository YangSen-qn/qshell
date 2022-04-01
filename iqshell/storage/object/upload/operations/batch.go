package operations

import (
	"github.com/qiniu/go-sdk/v7/storage"
	"github.com/qiniu/qshell/v2/iqshell"
	"github.com/qiniu/qshell/v2/iqshell/common/data"
	"github.com/qiniu/qshell/v2/iqshell/common/log"
	"github.com/qiniu/qshell/v2/iqshell/common/utils"
	"github.com/qiniu/qshell/v2/iqshell/common/workspace"
	"github.com/qiniu/qshell/v2/iqshell/storage/object/batch"
	"os"
	"path/filepath"
)

type BatchUploadInfo struct {
	GroupInfo        batch.Info
	UploadConfigFile string
	CallbackHost     string
	CallbackUrl      string
}

func (info *BatchUploadInfo) Check() *data.CodeError {
	if info.GroupInfo.WorkerCount < 1 || info.GroupInfo.WorkerCount > 2000 {
		info.GroupInfo.WorkerCount = 5
		log.WarningF("Tip: you can set <ThreadCount> value between 1 and 200 to improve speed, and now ThreadCount change to: %d",
			info.GroupInfo.Info.WorkerCount)
	}
	if err := info.GroupInfo.Check(); err != nil {
		return err
	}
	return nil
}

// BatchUpload 该命令会读取配置文件， 上传本地文件系统的文件到七牛存储中;
// 可以设置多线程上传，默认的线程区间在[iqshell.min_upload_thread_count, iqshell.max_upload_thread_count]
func BatchUpload(cfg *iqshell.Config, info BatchUploadInfo) {
	if iqshell.ShowDocumentIfNeeded(cfg) {
		return
	}

	if !iqshell.Check(cfg, iqshell.CheckAndLoadInfo{
		Checker: &info,
	}) {
		return
	}

	if len(info.UploadConfigFile) == 0 {
		log.Error("LocalDownloadConfig can't empty")
		return
	}

	upload2Info := BatchUpload2Info{
		BatchInfo:    info.GroupInfo,
		UploadConfig: DefaultUploadConfig(),
	}
	upload2Info.UploadConfig.Policy = &storage.PutPolicy{
		CallbackURL:  info.CallbackUrl,
		CallbackHost: info.CallbackHost,
	}

	if err := utils.UnMarshalFromFile(info.UploadConfigFile, &upload2Info.UploadConfig); err != nil {
		log.ErrorF("UnMarshal: read upload config error:%v config file:%s", err, info.UploadConfigFile)
		return
	}
	if err := utils.UnMarshalFromFile(info.UploadConfigFile, &cfg.CmdCfg.Log); err != nil {
		log.ErrorF("UnMarshal: read log setting error:%v config file:%s", err, info.UploadConfigFile)
		return
	}

	BatchUpload2(cfg, upload2Info)
}

type BatchUpload2Info struct {
	// 输入文件通过 upload config 输入
	BatchInfo batch.Info
	UploadConfig
}

func (info *BatchUpload2Info) Check() *data.CodeError {
	if info.BatchInfo.WorkerCount < 1 || info.BatchInfo.WorkerCount > 2000 {
		info.BatchInfo.WorkerCount = 5
		log.WarningF("Tip: you can set <ThreadCount> value between 1 and 200 to improve speed, and now ThreadCount change to: %d",
			info.BatchInfo.Info.WorkerCount)
	}
	if err := info.BatchInfo.Check(); err != nil {
		return err
	}
	if err := info.UploadConfig.Check(); err != nil {
		return err
	}
	return nil
}

func BatchUpload2(cfg *iqshell.Config, info BatchUpload2Info) {
	if shouldContinue := iqshell.CheckAndLoad(cfg, iqshell.CheckAndLoadInfo{
		Checker: &info,
		BeforeLogFile: func() {
			if len(info.RecordRoot) == 0 {
				info.RecordRoot = uploadCachePath(workspace.GetConfig(), &info.UploadConfig)
			}
			if data.Empty(cfg.CmdCfg.Log.LogFile) {
				workspace.GetConfig().Log.LogFile = data.NewString(filepath.Join(info.RecordRoot, "log.txt"))
			}
		},
	}); !shouldContinue {
		return
	}

	log.DebugF("record root: %s", info.RecordRoot)

	batchUpload(info)
}

func batchUpload(info BatchUpload2Info) {
	dbPath := filepath.Join(info.RecordRoot, ".ldb")
	log.InfoF("upload status db file path:%s", dbPath)

	// 扫描本地文件
	needScanLocal := false
	if data.Empty(info.FileList) {
		needScanLocal = true
	} else {
		if _, err := os.Stat(info.FileList); err == nil {
			// 存在 file list 无需再重新扫描
			needScanLocal = false
			info.BatchInfo.InputFile = info.FileList
		} else {
			info.BatchInfo.InputFile = filepath.Join(info.RecordRoot, ".cache")
			if _, statErr := os.Stat(info.BatchInfo.InputFile); statErr == nil {
				//file exists
				needScanLocal = info.IsRescanLocal()
			} else {
				needScanLocal = true
			}
		}
	}

	if needScanLocal {
		if data.Empty(info.SrcDir) {
			log.ErrorF("scan error: src dir was empty")
			return
		}

		if len(info.BatchInfo.InputFile) == 0 {
			info.BatchInfo.InputFile = filepath.Join(info.RecordRoot, ".cache")
		}

		_, err := utils.DirCache(info.SrcDir, info.BatchInfo.InputFile)
		if err != nil {
			log.ErrorF("create dir files cache error:%v", err)
			return
		}
	}

	batchUploadFlow(info, info.UploadConfig, dbPath)
}

func batchUploadFlow(info BatchUpload2Info, uploadConfig UploadConfig, dbPath string) {
	//exporter, err := export.NewFileExport(info.BatchInfo.FileExporterConfig)
	//if err != nil {
	//	log.Error(err)
	//	return
	//}
	//
	//mac, err := workspace.GetMac()
	//if err != nil {
	//	log.Error("get mac error:" + err.Error())
	//	return
	//}
	//
	//timeStart := time.Now()
	//syncLocker := synchronized.NewSynchronized(nil)
	//var totalFileCount = handler.Scanner().LineCount()
	//var currentFileCount int64
	//var successFileCount int64
	//var failureFileCount int64
	//var notOverwriteCount int64
	//var skippedFileCount int64
	//
	//f := &flow.Flow{}
	//// 配置 work provider
	//workCreator := flow.NewLineSeparateWorkCreator(info.BatchInfo.ItemSeparate, func(items []string) (work flow.Work, err *data.CodeError) {
	//	if len(items) < 3 {
	//		syncLocker.Do(func() { skippedFileCount += 1 })
	//		log.InfoF("Skip by invalid line, items should more than 2:%s", line)
	//		return nil, true
	//	}
	//	fileRelativePath := items[0]
	//
	//	//check skip local file or folder
	//	if skip, prefix := uploadConfig.HitByPathPrefixes(fileRelativePath); skip {
	//		log.InfoF("Skip by path prefix `%s` for local file path `%s`", prefix, fileRelativePath)
	//		syncLocker.Do(func() { skippedFileCount += 1 })
	//		return nil, true
	//	}
	//
	//	if skip, prefix := uploadConfig.HitByFilePrefixes(fileRelativePath); skip {
	//		log.InfoF("Skip by file prefix `%s` for local file path `%s`", prefix, fileRelativePath)
	//		syncLocker.Do(func() { skippedFileCount += 1 })
	//		return nil, true
	//	}
	//
	//	if skip, fixedStr := uploadConfig.HitByFixesString(fileRelativePath); skip {
	//		log.InfoF("Skip by fixed string `%s` for local file path `%s`", fixedStr, fileRelativePath)
	//		syncLocker.Do(func() { skippedFileCount += 1 })
	//		return nil, true
	//	}
	//
	//	if skip, suffix := uploadConfig.HitBySuffixes(fileRelativePath); skip {
	//		log.InfoF("Skip by suffix `%s` for local file `%s`", suffix, fileRelativePath)
	//		syncLocker.Do(func() { skippedFileCount += 1 })
	//		return nil, true
	//	}
	//
	//	//pack the upload file key
	//	fileSize, _ := strconv.ParseInt(items[1], 10, 64)
	//	modifyTime, _ := strconv.ParseInt(items[2], 10, 64)
	//	key := fileRelativePath
	//	//check ignore dir
	//	if uploadConfig.IsIgnoreDir() {
	//		key = filepath.Base(key)
	//	}
	//	//check prefix
	//	if data.NotEmpty(uploadConfig.KeyPrefix) {
	//		key = strings.Join([]string{uploadConfig.KeyPrefix, key}, "")
	//	}
	//	//convert \ to / under windows
	//	if utils.IsWindowsOS() {
	//		key = strings.Replace(key, "\\", "/", -1)
	//	}
	//	//check file encoding
	//	if data.NotEmpty(uploadConfig.FileEncoding) && utils.IsGBKEncoding(uploadConfig.FileEncoding) {
	//		key, _ = utils.Gbk2Utf8(key)
	//	}
	//	log.DebugF("Key:%s FileSize:%d ModifyTime:%d", key, fileSize, modifyTime)
	//
	//	localFilePath := filepath.Join(uploadConfig.SrcDir, fileRelativePath)
	//	apiInfo := &UploadInfo{
	//		ApiInfo: upload.ApiInfo{
	//			FilePath:         localFilePath,
	//			ToBucket:         uploadConfig.Bucket,
	//			SaveKey:          key,
	//			MimeType:         "",
	//			FileType:         uploadConfig.FileType,
	//			CheckExist:       uploadConfig.CheckExists,
	//			CheckHash:        uploadConfig.CheckHash,
	//			CheckSize:        uploadConfig.CheckSize,
	//			Overwrite:        uploadConfig.Overwrite,
	//			UpHost:           uploadConfig.UpHost,
	//			FileStatusDBPath: dbPath,
	//			TokenProvider:    nil,
	//			TryTimes:         3,
	//			TryInterval:      500 * time.Millisecond,
	//			FileSize:         fileSize,
	//			FileModifyTime:   modifyTime,
	//			DisableForm:      uploadConfig.DisableForm,
	//			DisableResume:    uploadConfig.DisableResume,
	//			UseResumeV2:      uploadConfig.ResumableAPIV2,
	//			ChunkSize:        uploadConfig.ResumableAPIV2PartSize,
	//			PutThreshold:     uploadConfig.PutThreshold,
	//			Progress:         nil,
	//		},
	//		Policy:          uploadConfig.Policy,
	//		DeleteOnSuccess: uploadConfig.DeleteOnSuccess,
	//	}
	//	apiInfo.TokenProvider = createTokenProviderWithMac(mac, apiInfo)
	//	return apiInfo, hasMore
	//})
	//if provider, e := flow.NewWorkProviderOfFile(info.BatchInfo.InputFile, info.BatchInfo.EnableStdin, workCreator); e != nil {
	//	return
	//} else {
	//	f.WorkProvider = provider
	//}
	//
	//// 配置 worker provider
	//f.WorkerProvider = flow.NewWorkerProvider(func() (flow.Worker, *data.CodeError) {
	//	return flow.NewWorker(func(work flow.Work) (flow.Result, *data.CodeError) {
	//		in := work.(PrivateUrlInfo)
	//		if deadline, e := in.getDeadlineOfInt(); e != nil {
	//			return nil, e
	//		} else {
	//			return download.PublicUrlToPrivate(download.PublicUrlToPrivateApiInfo{
	//				PublicUrl: in.PublicUrl,
	//				Deadline:  deadline,
	//			})
	//		}
	//	}), nil
	//})
	//
	//// 配置时间监听
	//f.EventListener = flow.EventListener{
	//	WillWorkFunc:   nil,
	//	OnWorkSkipFunc: nil,
	//	OnWorkSuccessFunc: func(work flow.Work, result flow.Result) {
	//		url := result.(string)
	//		log.Alert(url)
	//	},
	//	OnWorkFailFunc: func(work flow.Work, err *data.CodeError) {
	//		log.Error(err)
	//	},
	//}
	//
	//// 开始
	//f.Start()
	//work.NewFlowHandler(info.BatchInfo.FlowInfo).ReadWork(func() (work work.Work, hasMore bool) {
	//	line, hasMore := handler.Scanner().ScanLine()
	//	if !hasMore {
	//		return
	//	}
	//
	//	if len(line) == 0 {
	//		syncLocker.Do(func() { skippedFileCount += 1 })
	//		log.InfoF("Skip by invalid line, items should more than 2:%s", line)
	//		return
	//	}
	//
	//	items := utils.SplitString(line, info.BatchInfo.ItemSeparate)
	//	if len(items) < 3 {
	//		syncLocker.Do(func() { skippedFileCount += 1 })
	//		log.InfoF("Skip by invalid line, items should more than 2:%s", line)
	//		return nil, true
	//	}
	//	fileRelativePath := items[0]
	//
	//	//check skip local file or folder
	//	if skip, prefix := uploadConfig.HitByPathPrefixes(fileRelativePath); skip {
	//		log.InfoF("Skip by path prefix `%s` for local file path `%s`", prefix, fileRelativePath)
	//		syncLocker.Do(func() { skippedFileCount += 1 })
	//		return nil, true
	//	}
	//
	//	if skip, prefix := uploadConfig.HitByFilePrefixes(fileRelativePath); skip {
	//		log.InfoF("Skip by file prefix `%s` for local file path `%s`", prefix, fileRelativePath)
	//		syncLocker.Do(func() { skippedFileCount += 1 })
	//		return nil, true
	//	}
	//
	//	if skip, fixedStr := uploadConfig.HitByFixesString(fileRelativePath); skip {
	//		log.InfoF("Skip by fixed string `%s` for local file path `%s`", fixedStr, fileRelativePath)
	//		syncLocker.Do(func() { skippedFileCount += 1 })
	//		return nil, true
	//	}
	//
	//	if skip, suffix := uploadConfig.HitBySuffixes(fileRelativePath); skip {
	//		log.InfoF("Skip by suffix `%s` for local file `%s`", suffix, fileRelativePath)
	//		syncLocker.Do(func() { skippedFileCount += 1 })
	//		return nil, true
	//	}
	//
	//	//pack the upload file key
	//	fileSize, _ := strconv.ParseInt(items[1], 10, 64)
	//	modifyTime, _ := strconv.ParseInt(items[2], 10, 64)
	//	key := fileRelativePath
	//	//check ignore dir
	//	if uploadConfig.IsIgnoreDir() {
	//		key = filepath.Base(key)
	//	}
	//	//check prefix
	//	if data.NotEmpty(uploadConfig.KeyPrefix) {
	//		key = strings.Join([]string{uploadConfig.KeyPrefix, key}, "")
	//	}
	//	//convert \ to / under windows
	//	if utils.IsWindowsOS() {
	//		key = strings.Replace(key, "\\", "/", -1)
	//	}
	//	//check file encoding
	//	if data.NotEmpty(uploadConfig.FileEncoding) && utils.IsGBKEncoding(uploadConfig.FileEncoding) {
	//		key, _ = utils.Gbk2Utf8(key)
	//	}
	//	log.DebugF("Key:%s FileSize:%d ModifyTime:%d", key, fileSize, modifyTime)
	//
	//	localFilePath := filepath.Join(uploadConfig.SrcDir, fileRelativePath)
	//	apiInfo := &UploadInfo{
	//		ApiInfo: upload.ApiInfo{
	//			FilePath:         localFilePath,
	//			ToBucket:         uploadConfig.Bucket,
	//			SaveKey:          key,
	//			MimeType:         "",
	//			FileType:         uploadConfig.FileType,
	//			CheckExist:       uploadConfig.CheckExists,
	//			CheckHash:        uploadConfig.CheckHash,
	//			CheckSize:        uploadConfig.CheckSize,
	//			Overwrite:        uploadConfig.Overwrite,
	//			UpHost:           uploadConfig.UpHost,
	//			FileStatusDBPath: dbPath,
	//			TokenProvider:    nil,
	//			TryTimes:         3,
	//			TryInterval:      500 * time.Millisecond,
	//			FileSize:         fileSize,
	//			FileModifyTime:   modifyTime,
	//			DisableForm:      uploadConfig.DisableForm,
	//			DisableResume:    uploadConfig.DisableResume,
	//			UseResumeV2:      uploadConfig.ResumableAPIV2,
	//			ChunkSize:        uploadConfig.ResumableAPIV2PartSize,
	//			PutThreshold:     uploadConfig.PutThreshold,
	//			Progress:         nil,
	//		},
	//		Policy:          uploadConfig.Policy,
	//		DeleteOnSuccess: uploadConfig.DeleteOnSuccess,
	//	}
	//	apiInfo.TokenProvider = createTokenProviderWithMac(mac, apiInfo)
	//	return apiInfo, hasMore
	//}).DoWork(func(work work.Work) (work.Result, *data.CodeError) {
	//	syncLocker.Do(func() {
	//		currentFileCount += 1
	//	})
	//	apiInfo := work.(*UploadInfo)
	//
	//	log.AlertF("Uploading %s [%d/%d, %.1f%%] ...", apiInfo.FilePath, currentFileCount, totalFileCount,
	//		float32(currentFileCount)*100/float32(totalFileCount))
	//
	//	res, err := uploadFile(apiInfo)
	//
	//	if err != nil {
	//		return nil, err
	//	}
	//	return res, nil
	//}).OnWorkResult(func(work work.Work, result work.Result) {
	//	apiInfo := work.(*UploadInfo)
	//	res := result.(upload.ApiResult)
	//	if res.IsOverWrite {
	//		exporter.Override().ExportF("upload overwrite, %s => [%s:%s]", apiInfo.FilePath, apiInfo.ToBucket, apiInfo.SaveKey)
	//	} else if res.IsSkip {
	//
	//	} else {
	//		exporter.Success().ExportF("upload success, %s => [%s:%s]", apiInfo.FilePath, apiInfo.ToBucket, apiInfo.SaveKey)
	//	}
	//
	//	syncLocker.Do(func() {
	//		if res.IsNotOverWrite {
	//			notOverwriteCount += 1
	//		} else if res.IsSkip {
	//			skippedFileCount += 1
	//		} else {
	//			successFileCount += 1
	//		}
	//	})
	//}).OnWorkError(func(work work.Work, err *data.CodeError) {
	//	syncLocker.Do(func() {
	//		failureFileCount += 1
	//	})
	//
	//	apiInfo := work.(*UploadInfo)
	//	exporter.Fail().ExportF("%s%s%ld%s%s%s%ld%s error:%s", /* path fileSize fileModifyTime */
	//		apiInfo.FilePath, info.BatchInfo.ItemSeparate,
	//		apiInfo.FileSize, info.BatchInfo.ItemSeparate,
	//		apiInfo.FileModifyTime, info.BatchInfo.ItemSeparate,
	//		err)
	//}).Start()
	//
	//log.Alert("--------------- Upload Result ---------------")
	//log.AlertF("%20s%10d", "Total:", totalFileCount)
	//log.AlertF("%20s%10d", "Success:", successFileCount)
	//log.AlertF("%20s%10d", "Failure:", failureFileCount)
	//log.AlertF("%20s%10d", "NotOverwrite:", notOverwriteCount)
	//log.AlertF("%20s%10d", "Skipped:", skippedFileCount)
	//log.AlertF("%20s%15s", "Duration:", time.Since(timeStart))
	//log.AlertF("---------------------------------------------")
	//if workspace.GetConfig().Log.Enable() {
	//	log.AlertF("See upload log at path:%s \n\n", workspace.GetConfig().Log.LogFile.Value())
	//}
}

type BatchUploadConfigMouldInfo struct {
}

func BatchUploadConfigMould(cfg *iqshell.Config, info BatchUploadConfigMouldInfo) {
	log.Alert(uploadConfigMouldJsonString)
}
