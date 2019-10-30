package main

import (
	"fmt"
	"github.com/go-redis/redis"
	"gopkg.in/telegram-bot-api.v4"
	"strconv"
	"strings"
)

const (
	// token from bot_father
	BotToken = ""
	// name to handle messages from chats (like /help@bot_name)
	BotName               = ""
	GenerateTokenAttempts = 3

	// error messages section
	ErrorMessageInternalError    = "Try again later"
	ErrorMessageTokenRevoked     = "Token for your chat was revoked. Set new one"
	ErrorMessageTokenNotSet      = "Set token from /generate command first"
	ErrorMessageParam            = "Check params"
	ErrorMessageWrongChatPrivate = "Command is available only in group chat"
	ErrorMessageWrongChatGroup   = "Command is available only in private chat"
	ErrorMessageSend             = "Could not sent message to your target chat. Try later or set new chat"
	ErrorMessageNeedRevokeToken  = "Revoke token in your target chat"

	// success messages section
	MessageOK = "Success"
)

var redisClient *redis.Client

func sendMessage(bot *tgbotapi.BotAPI, chatID int64, message string) bool {
	_, err := bot.Send(tgbotapi.NewMessage(
		chatID,
		message))

	if err != nil {
		fmt.Println(err)
		return false
	}

	return true
}

func handleCommand(bot *tgbotapi.BotAPI, chat *tgbotapi.Chat, userID int, text string) {
	params := strings.Split(text, " ")
	command := params[0]
	println(text)

	switch command {
	// Set target chat for user by generated token of chat
	case "/set":
		fallthrough
	case "/set@" + BotName:
		{
			if !chat.IsPrivate() {
				sendMessage(bot, chat.ID, ErrorMessageWrongChatGroup)
				return
			}

			if len(params) < 2 {
				sendMessage(bot, chat.ID, ErrorMessageParam)
				return
			}
			token := params[1]

			_, err := getChatIDByToken(token)
			if err == redis.Nil {
				sendMessage(bot, chat.ID, "Chat with such token does not exist!")
				return
			}

			_, err = setTokenByUserID(userID, token)
			if err != nil {
				sendMessage(bot, chat.ID, ErrorMessageInternalError)
				return
			}

			sendMessage(bot, chat.ID, MessageOK)
		}
	// Get token of chat, which was specified by user earlier
	case "/get":
		fallthrough
	case "/get@" + BotName:
		{
			if !chat.IsPrivate() {
				sendMessage(bot, chat.ID, ErrorMessageWrongChatGroup)
				return
			}

			token, err := getTokenByUserID(userID)
			if err == redis.Nil {
				sendMessage(bot, chat.ID, "You should set token first")
				return
			}

			res, err := getChatIDByToken(token)
			if err != nil {
				fmt.Println(err)
				if err == redis.Nil {
					sendMessage(bot, chat.ID, ErrorMessageTokenRevoked)
				} else {
					sendMessage(bot, chat.ID, ErrorMessageInternalError)
				}
				return
			}

			chatID, err := strconv.ParseInt(res, 10, 64)
			if err != nil {
				sendMessage(bot, chat.ID, ErrorMessageInternalError)
				return
			}

			msgChat, err := bot.GetChat(tgbotapi.ChatConfig{
				ChatID:             chatID,
				SuperGroupUsername: "",
			})
			if err != nil {
				sendMessage(bot, chat.ID, ErrorMessageInternalError)
				return
			}

			sendMessage(bot, chat.ID, "Selected chat: "+msgChat.Title)
		}
	// Generate token for chat
	case "/generate":
		fallthrough
	case "/generate@" + BotName:
		{
			if !chat.IsGroup() && !chat.IsSuperGroup() {
				sendMessage(bot, chat.ID, ErrorMessageWrongChatPrivate)
				return
			}

			oldToken, err := getTokenByChatID(chat.ID)
			if err != nil && err != redis.Nil {
				sendMessage(bot, chat.ID, ErrorMessageInternalError)
				return
			}
			_, err = deleteChatIDByToken(oldToken)
			if err != nil {
				sendMessage(bot, chat.ID, ErrorMessageInternalError)
				return
			}

			token := ""
			attempts := 0
			for {
				if attempts > GenerateTokenAttempts {
					break
				}

				token = generateToken()
				_, err := getChatIDByToken(token)
				if err == redis.Nil {
					break
				}
				attempts++
			}

			if attempts > GenerateTokenAttempts {
				sendMessage(bot, chat.ID, ErrorMessageInternalError)
				return
			}

			_, err = setTokenByChatID(token, chat.ID)
			if err != nil {
				sendMessage(bot, chat.ID, ErrorMessageInternalError)
				return
			}

			_, err = setChatIDByToken(token, chat.ID)
			if err != nil {
				sendMessage(bot, chat.ID, ErrorMessageInternalError)
				return
			}

			sendMessage(bot, chat.ID, "Your token was generated, forward following message in private messages with me : ")
			sendMessage(bot, chat.ID, "/set "+token)
		}
	case "/help":
		fallthrough
	case "/help@" + BotName:
		sendMessage(bot, chat.ID, "Hi! I can send messages anonymously. Add me to any chat, generate token and set it in chat with me.\n"+
			"Available commands: \n"+
			"/set - set your chat by it's token\n"+
			"/get - info about your target chat\n"+
			"/generate - associate (or revoke) new token with current chat")
	default:
		sendMessage(bot, chat.ID, "command not exists")
	}
}

func handleMessage(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	chatID := message.Chat.ID
	userID := message.From.ID
	text := message.Text
	if userID == 0 || chatID == 0 || len(text) == 0{
		return
	}

	if text[0] == '/' {
		handleCommand(bot, message.Chat, userID, text)
		return
	}

	token, err := getTokenByUserID(userID)
	if err != nil {
		if err == redis.Nil {
			sendMessage(bot, chatID, ErrorMessageInternalError)
		} else {
			sendMessage(bot, chatID, ErrorMessageTokenNotSet)
		}
		return
	}

	res, err := getChatIDByToken(token)
	if err == redis.Nil {
		sendMessage(bot, chatID, ErrorMessageNeedRevokeToken)
		return
	}
	targetChatID, err := strconv.ParseInt(res, 10, 64)
	if err != nil {
		sendMessage(bot, chatID, ErrorMessageInternalError)
		return
	}

	if sendMessage(bot, targetChatID, message.Text) {
		sendMessage(bot, chatID, MessageOK)
	} else {
		sendMessage(bot, chatID, ErrorMessageSend)
	}
}

func main() {
	bot, err := tgbotapi.NewBotAPI(BotToken)
	if err != nil {
		panic(err)
	}

	redisClient = redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	fmt.Printf("Auth on bot %s\n", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates, err := bot.GetUpdatesChan(u)

	for update := range updates {
		message := update.Message

		if message == nil {
			continue
		}

		if message.From.ID == 0 {
			fmt.Println("no ID")
			continue
		}

		handleMessage(bot, message)
	}
}
