package server

import "strings"

func IsGate(appId string) bool {
	return strings.HasPrefix(appId, GATE_SVC)
}

func IsActor(appId string) bool {
	return strings.HasPrefix(appId, ACTOR_SVC)
}

func IsIDIP(appId string) bool {
	return strings.HasPrefix(appId, IDIP_SVC)
}

func IsCenter(appId string) bool {
	return strings.HasPrefix(appId, CENTER_SVC)
}
