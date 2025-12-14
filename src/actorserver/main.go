package main

import (
	"errors"
	"gitee.com/bychannel/aniwar/src/actorserver/allianceactor"
	"gitee.com/bychannel/aniwar/src/actorserver/mailactor"

	"gitee.com/aniwar2/musae/framework/base"
	"gitee.com/aniwar2/musae/framework/global"
	"gitee.com/aniwar2/musae/framework/logger"
	"gitee.com/aniwar2/musae/framework/process"
	"gitee.com/bychannel/aniwar/src/actorserver/centeractor"
	"gitee.com/bychannel/aniwar/src/actorserver/frame"
	"gitee.com/bychannel/aniwar/src/actorserver/roomactor"
	"gitee.com/bychannel/aniwar/src/actorserver/useractor"
)

func InitActorFactory(srv base.IServer) error {
	actors := srv.GetActors()
	var factory []base.FActorFactory
	logger.Debugf("main.actors, %v", actors)
	for _, actor := range actors {
		switch actor {
		case global.UserActorType:
			factory = append(factory, useractor.New)
		case global.RoomActorType:
			factory = append(factory, roomactor.New)
		case global.AllianceActorType:
			factory = append(factory, allianceactor.New)
		case global.CenterActorType:
			factory = append(factory, centeractor.New)
		case global.MailActorType:
			factory = append(factory, mailactor.New)
		default:
			return errors.New("unknown actor type")
		}
	}
	logger.Debugf("main.ActorFactory, %v", factory)
	srv.RegisterActorFactory(factory...)
	return nil
}

func main() {
	process.Start(frame.NewActorServer(), InitActorFactory)
}
