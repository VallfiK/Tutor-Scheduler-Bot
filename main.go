package main

import (
	"database/sql"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

var (
	bot           *tgbotapi.BotAPI
	db            *sql.DB
	lastMessageID map[int64]int // Карта: ChatID -> MessageID
)

func main() {
	lastMessageID = make(map[int64]int)

	var err error
	db, err = InitDB()
	if err != nil {
		panic("Ошибка инициализации базы данных: " + err.Error())
	}
	defer db.Close()

	botToken := "ВашТокенБота"
	if botToken == "" {
		panic("Токен бота не указан. Установите переменную окружения TELEGRAM_BOT_TOKEN")
	}

	bot, err = tgbotapi.NewBotAPI(botToken)
	if err != nil {
		panic("Ошибка инициализации бота: " + err.Error())
	}

	bot.Debug = true

	StartNotificationScheduler()

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	_, err = bot.RemoveWebhook()
	if err != nil {
		panic("Ошибка при снятии вебхука: " + err.Error())
	}

	updates, err := bot.GetUpdatesChan(u)
	if err != nil {
		panic("Ошибка получения обновлений: " + err.Error())
	}

	for update := range updates {
		if update.UpdateID != 0 {
			// Лог удален
		}
		handleUpdate(update)
	}

	select {
	case <-signalChan:
		// Лог удален
		return
	}
}

func handleUpdate(update tgbotapi.Update) {
	// Лог удален
	if update.Message != nil {
		deleteMessage(update.Message.Chat.ID, update.Message.MessageID)
	}

	if update.CallbackQuery != nil {
		handleCallback(update.CallbackQuery)
		return
	}

	if update.Message == nil {
		return
	}

	if update.Message.IsCommand() {
		handleCommand(update.Message)
	}
}

func handleCommand(msg *tgbotapi.Message) {
	switch msg.Command() {
	case "start":
		handleStart(msg)
	case "schedule":
		handleTeacherSchedule(msg.Chat.ID)
	case "students":
		handleTeacherStudents(msg.Chat.ID)
	case "book":
		handleStudentBook(msg.Chat.ID)
	case "mybookings":
		handleStudentBookings(msg.Chat.ID)
	case "cancel":
		handleStudentCancel(msg.Chat.ID)
	default:
		// Неизвестная команда — показываем меню
		user, err := getUser(msg.Chat.ID)
		if err != nil {
			sendMessage(msg.Chat.ID, "Ошибка получения данных пользователя.")
			return
		}
		if user.Role == "teacher" {
			showTeacherMenu(msg.Chat.ID)
		} else {
			showStudentMenu(msg.Chat.ID)
		}
	}
}

func deleteMessage(chatID int64, messageID int) {
	deleteMsg := tgbotapi.NewDeleteMessage(chatID, messageID)
	if _, err := bot.Send(deleteMsg); err != nil {
		fmt.Println("Ошибка удаления сообщения:", err) // Логирование
	}
}

// main.go
func handleMessage(msg *tgbotapi.Message) {
	// Обработка текстовых сообщений (например, добавление слота)
	if isValidDateTimeFormat(msg.Text) {
		parts := strings.Split(msg.Text, " ")
		if len(parts) == 2 {
			startTime := parts[0]
			endTime := parts[1]
			err := addScheduleSlot(msg.Chat.ID, startTime, endTime)
			if err != nil {
				sendMessage(msg.Chat.ID, "Ошибка добавления слота.")
			} else {
				sendMessage(msg.Chat.ID, "Слот успешно добавлен!")
			}
		} else {
			sendMessage(msg.Chat.ID, "Неверный формат. Используйте: ГГГГ-ММ-ДДTЧЧ:ММ:ССZ ГГГГ-ММ-ДДTЧЧ:ММ:ССZ")
		}
	} else {
		sendMessage(msg.Chat.ID, "Неверный формат даты и времени. Используйте: ГГГГ-ММ-ДДTЧЧ:ММ:ССZ ГГГГ-ММ-ДДTЧЧ:ММ:ССZ")
	}
}

// Новая функция для проверки формата даты и времени
func isValidDateTimeFormat(text string) bool {
	parts := strings.Split(text, " ")
	if len(parts) != 2 {
		return false
	}
	for _, part := range parts {
		if !strings.HasPrefix(part, strings.Join(strings.Split(time.Now().Format("2006-01-02"), "-"), "-")) &&
			!isValidTime(part) {
			return false
		}
	}
	return true
}
