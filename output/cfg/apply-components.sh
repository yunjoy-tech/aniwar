#!/usr/bin/env bash
kubectl apply \
	-f ./k8s-config.yaml \
	-f ./component/binding.yaml \
	-f ./component/cron-hour.yaml \
	-f ./component/mongo-account.yaml \
	-f ./component/mongo-game.yaml \
	-f ./component/mongo-mail.yaml \
	-f ./component/redis-global.yaml \
	-f ./component/redis-lock.yaml \
	-f ./component/redis-cache.yaml \
	-f ./component/pubsub-appid.yaml \
	-f ./component/pubsub-global.yaml \
	-f ./component/pubsub-private.yaml \
	-f ./component/resiliency.yaml \
	-n aniwar
