package flow

import (
	"fmt"
	"github.com/qiniu/qshell/v2/iqshell/common/data"
	"github.com/qiniu/qshell/v2/iqshell/common/utils"
)

const DefaultLineItemSeparate = "\t"

type lineSeparateWorkCreator struct {
	separate      string
	minItemsCount int
	creatorFunc   func(items []string) (work Work, err *data.CodeError)
}

func (l *lineSeparateWorkCreator) Create(info string) (work Work, err *data.CodeError) {
	items := utils.SplitString(info, l.separate)
	if len(items) >= l.minItemsCount {
		return l.creatorFunc(items)
	}
	return nil, data.NewError(data.ErrorCodeParamMissing, fmt.Sprintf("%s%serror:at least %d parameter is required", info, l.separate, l.minItemsCount))
}

func NewLineSeparateWorkCreator(separate string, minItemsCount int, creatorFunc func(items []string) (work Work, err *data.CodeError)) WorkCreator {
	if len(separate) == 0 {
		separate = DefaultLineItemSeparate
	}
	return &lineSeparateWorkCreator{
		separate:      separate,
		minItemsCount: minItemsCount,
		creatorFunc:   creatorFunc,
	}
}
