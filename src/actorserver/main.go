package main

import (
	"errors"
	"gitee.com/aniwar2/aniwar/src/actorserver/allianceactor"
	"gitee.com/aniwar2/aniwar/src/actorserver/mailactor"

	"gitee.com/aniwar2/aniwar/src/actorserver/centeractor"
	"gitee.com/aniwar2/aniwar/src/actorserver/frame"
	"gitee.com/aniwar2/aniwar/src/actorserver/roomactor"
	"gitee.com/aniwar2/aniwar/src/actorserver/useractor"
	"gitee.com/aniwar2/musae/base"
	"gitee.com/aniwar2/musae/global"
	"gitee.com/aniwar2/musae/logger"
	"gitee.com/aniwar2/musae/process"
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
