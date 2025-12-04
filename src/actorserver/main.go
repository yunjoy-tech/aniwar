package main

import (
	"errors"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/actorserver/allianceactor"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/actorserver/mailactor"

	"gitlab.musadisca-games.com/wangxw/aniwar/src/actorserver/centeractor"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/actorserver/frame"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/actorserver/roomactor"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/actorserver/useractor"
	"gitlab.musadisca-games.com/wangxw/musae/framework/base"
	"gitlab.musadisca-games.com/wangxw/musae/framework/global"
	"gitlab.musadisca-games.com/wangxw/musae/framework/logger"
	"gitlab.musadisca-games.com/wangxw/musae/framework/process"
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
