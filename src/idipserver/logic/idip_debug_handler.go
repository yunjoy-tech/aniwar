package logic

//func (s *IDIPServer) cfgReload(ctx context.Context, in *common.InvocationEvent) (out *common.Content, err error) {
//	defer func() {
//		if err := recover(); err != any(nil) {
//			logger.Trace("configReload failed, err: ", err)
//		}
//	}()
//
//	out = &common.Content{
//		ContentType: in.ContentType,
//		DataTypeURL: in.DataTypeURL,
//	}
//
//	if in == nil {
//		err = fmt.Errorf("nil invocation parameter")
//		logger.Warn("cfgReload nil invocation parameter")
//		return out, err
//	}
//	logger.Debugf("[idip] excelReload - ContentType:%s, Verb:%s, QueryString:%s, Data:%v", in.ContentType, in.Verb, in.QueryString, in.Data)
//
//	param := &comn.ReloadParam{}
//	err = json.Unmarshal(in.Data, param)
//	if err != nil {
//		logger.Warn("cfgReload ReloadParam error")
//		return out, err
//	}
//
//	switch param.Type {
//	case "conf":
//		s.SaveToConfigCenter(db.KeyCfgReloadConf, time.Now().Local().String())
//	case "excel":
//		s.SaveToConfigCenter(db.KeyCfgReloadExcel, param.Files)
//	default:
//		logger.Warnf("cfgReload ReloadParam error,param:[%+v]", param)
//		out.Data = s.GenRet("cfgReload ReloadParam error")
//		return out, err
//	}
//
//	out.Data = s.GenRet("active success")
//	logger.Debugf("[idip] configReload , out: %s", string(out.Data))
//	return out, nil
//
//}
