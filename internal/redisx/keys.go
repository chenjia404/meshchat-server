package redisx

import "fmt"

func ChallengeKey(challengeID string) string {
	return fmt.Sprintf("chat:auth:challenge:%s", challengeID)
}

func CooldownKey(groupID string, userID uint64) string {
	return fmt.Sprintf("chat:cooldown:%s:%d", groupID, userID)
}

func OnlineUserKey(userID uint64) string {
	return fmt.Sprintf("chat:online:user:%d", userID)
}

func OnlineGroupKey(groupID string, userID uint64) string {
	return fmt.Sprintf("chat:online:group:%s:%d", groupID, userID)
}

func GroupEventsChannel(groupID string) string {
	return fmt.Sprintf("chat:events:group:%s", groupID)
}

func DMEventsChannel(conversationID string) string {
	return fmt.Sprintf("chat:events:dm:%s", conversationID)
}
