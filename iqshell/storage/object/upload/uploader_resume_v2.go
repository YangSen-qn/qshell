package upload

import (
	"errors"
	"github.com/qiniu/go-sdk/v7/storage"
	"github.com/qiniu/qshell/v2/iqshell/common/workspace"
	"os"
)

type resumeV2Uploader struct {
	cfg *storage.Config
	ext *storage.RputV2Extra
}

func newResumeV2Uploader(cfg *storage.Config, ext *storage.RputV2Extra) Uploader {
	return &resumeV2Uploader{
		cfg: cfg,
		ext: ext,
	}
}

func (r *resumeV2Uploader) upload(info ApiInfo) (ret UploadResult, err error) {
	file, err := os.Open(info.FilePath)
	if err != nil {
		err = errors.New("resume v2 upload: open file error:" + err.Error())
		return
	}

	fileStatus, err := file.Stat()
	if err != nil {
		err = errors.New("resume v2 upload: ger file status error:" + err.Error())
		return
	}

	up := storage.NewResumeUploaderV2(r.cfg)
	err = up.Put(workspace.GetContext(), &ret, info.TokenProvider(), info.SaveKey, file, fileStatus.Size(), r.ext)
	if err != nil {
		err = errors.New("resume v2 upload: upload error:" + err.Error())
	}

	return
}