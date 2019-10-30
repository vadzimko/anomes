package main

import (
	"math/rand"
	"strconv"
)

const (
	redisKeyChatIDByTokenPrefix       = "cibt_"
	redisKeyTokenByChatIdPrefix       = "tbci_"
	redisKeyChatIDByUserIDPrefix      = "cibui_"
)

func getChatIDByTokenKey(token string) string {
	return redisKeyChatIDByTokenPrefix + token
}

func getTokenByChatIDKey(chatID int64) string {
	return redisKeyTokenByChatIdPrefix + strconv.FormatInt(chatID, 10)
}

func getTokenByUserIDKey(userID int) string {
	return redisKeyChatIDByUserIDPrefix + strconv.Itoa(userID)
}

func getChatIDByToken(token string) (string, error) {
	key := getChatIDByTokenKey(token)
	return redisClient.Get(key).Result()
}

func deleteChatIDByToken(token string) (int64, error) {
	key := getChatIDByTokenKey(token)
	return redisClient.Del(key).Result()
}

func setChatIDByToken(token string, chatID int64) (string, error) {
	key := getChatIDByTokenKey(token)
	return redisClient.Set(key, chatID, 0).Result()
}

func getTokenByChatID(chatID int64) (string, error) {
	key := getTokenByChatIDKey(chatID)
	return redisClient.Get(key).Result()
}

func setTokenByChatID(token string, chatID int64) (string, error) {
	key := getTokenByChatIDKey(chatID)
	return redisClient.Set(key, token, 0).Result()
}

func getTokenByUserID(userID int) (string, error) {
	key := getTokenByUserIDKey(userID)
	return redisClient.Get(key).Result()
}

func setTokenByUserID(userID int, token string) (string, error) {
	key := getTokenByUserIDKey(userID)
	return redisClient.Set(key, token, 0).Result()
}

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

func generateToken() string {
	b := make([]rune, 32)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}
