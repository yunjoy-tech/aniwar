package main

import (
	"errors"
	"github.com/yunjoy-tech/aniwar/src/actorserver/allianceactor"
	"github.com/yunjoy-tech/aniwar/src/actorserver/centeractor"
	"github.com/yunjoy-tech/aniwar/src/actorserver/frame"
	"github.com/yunjoy-tech/aniwar/src/actorserver/mailactor"
	"github.com/yunjoy-tech/aniwar/src/actorserver/roomactor"
	"github.com/yunjoy-tech/aniwar/src/actorserver/useractor"
	"github.com/yunjoy-tech/aniwar/src/common/actor/stub"
	"github.com/yunjoy-tech/musae/base"
	"github.com/yunjoy-tech/musae/logger"
	"github.com/yunjoy-tech/musae/process"
)

func InitActorFactory(srv base.IServer) error {
	actors := srv.GetActors()
	var factory []base.FActorFactory
	logger.Debugf("main.actors, %v", actors)
	for _, actor := range actors {
		switch actor {
		case stub.UserActorType:
			factory = append(factory, useractor.New)
		case stub.RoomActorType:
			factory = append(factory, roomactor.New)
		case stub.AllianceActorType:
			factory = append(factory, allianceactor.New)
		case stub.CenterActorType:
			factory = append(factory, centeractor.New)
		case stub.MailActorType:
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
