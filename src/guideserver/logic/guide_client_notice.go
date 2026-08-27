package logic

import (
	"context"
	"fmt"
	"github.com/dapr/go-sdk/service/common"
	"github.com/yunjoy-tech/aniwar/src/common/conf"
	"github.com/yunjoy-tech/aniwar/src/common/db"
	"github.com/yunjoy-tech/musae/logger"
	"time"
)

func (s *GuideServer) Notice(ctx context.Context, in *common.InvocationEvent) (out *common.Content, err error) {
	defer func() {
		if err := recover(); err != any(nil) {
			logger.Error("/api/notice failed, err: ", err)
		}
	}()
	startTime := time.Now()
	out = &common.Content{
		ContentType: in.ContentType,
		DataTypeURL: in.DataTypeURL,
	}
	out.Data, err = s.getNoticeInfo(in)
	if err != nil {
		return out, err
	}
	delayTime := time.Since(startTime).Milliseconds()
	logger.Debugf("Notice Delay:%d Data:%s", delayTime, string(out.Data))
	return out, nil
}

func (s *GuideServer) getNoticeInfo(in *common.InvocationEvent) ([]byte, error) {
	if in == nil || len(in.Data) > conf.Base().GateMsgMaxSize {
		return nil, fmt.Errorf("invocation parameter error")
	}

	// noticeList := make([]*comn.Notice, 0)
	// 获取当前公告

	notice, err := s.GetConfigKeyForStr(db.KeyCfgServerNotice)
	/*data, err := json.Marshal(noticeList)
	if err != nil {
		logger.Warn("[guide] getNoticeInfo Marshal error, %v noticeList %+v", err, noticeList)
	}*/
	return []byte(notice), err
}
