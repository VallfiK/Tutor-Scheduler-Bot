package main

import (
	"fmt"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

// Отправка текстового сообщения
func sendMessage(chatID int64, text string) {
	// Удаляем предыдущее сообщение бота, если оно есть
	if lastID, exists := lastMessageID[chatID]; exists {
		deleteMessage(chatID, lastID)
	}

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	newMsg, err := bot.Send(msg)
	if err != nil {
		return
	}
	lastMessageID[chatID] = newMsg.MessageID // Сохраняем новый ID сообщения
}

// Отправка сообщения с клавиатурой
func sendMessageWithKeyboard(chatID int64, text string, keyboard *tgbotapi.InlineKeyboardMarkup) {
	if lastID, exists := lastMessageID[chatID]; exists {
		deleteMessage(chatID, lastID)
		fmt.Println("Deleted previous message ID:", lastID, "for chatID:", chatID)
	}

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	if keyboard != nil {
		msg.ReplyMarkup = keyboard
	}

	newMsg, err := bot.Send(msg)
	if err != nil {
		fmt.Println("Ошибка отправки сообщения с клавиатурой:", err, "chatID:", chatID)
		return
	}
	lastMessageID[chatID] = newMsg.MessageID
	fmt.Println("Sent new message ID:", newMsg.MessageID, "for chatID:", chatID)
}

func updateMessageWithKeyboard(chatID int64, text string, keyboard *tgbotapi.InlineKeyboardMarkup) {
	if lastID, exists := lastMessageID[chatID]; exists {
		// Обновляем текст
		editMsg := tgbotapi.NewEditMessageText(chatID, lastID, text)
		editMsg.ParseMode = "Markdown"
		bot.Send(editMsg)

		// Обновляем клавиатуру
		if keyboard != nil {
			editMarkup := tgbotapi.NewEditMessageReplyMarkup(chatID, lastID, *keyboard)
			bot.Send(editMarkup)
		}
	} else {
		sendMessageWithKeyboard(chatID, text, keyboard)
	}
}

// Отправка сообщения с inline-клавиатурой
func SendInlineKeyboard(chatID int64, text string, buttons []tgbotapi.InlineKeyboardButton) {
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(buttons...),
	)
	sendMessageWithKeyboard(chatID, text, &keyboard)
}

// Форматирование длительности
func FormatDuration(start, end string) string {
	startTime, err1 := time.Parse(time.RFC3339, start)
	endTime, err2 := time.Parse(time.RFC3339, end)
	if err1 != nil || err2 != nil {
		return fmt.Sprintf("%s - %s", start, end)
	}
	return fmt.Sprintf("%s - %s", startTime.Format("15:04"), endTime.Format("15:04"))
}

// Проверка, является ли пользователь учителем
func IsTeacher(telegramID int64) bool {
	user, err := getUser(telegramID)
	if err != nil {
		return false
	}
	return user.Role == "teacher"
}

// Проверка, является ли пользователь учеником
func IsStudent(telegramID int64) bool {
	user, err := getUser(telegramID)
	if err != nil {
		return false
	}
	return user.Role == "student"
}

// Генерация текста для отображения расписания
func FormatSchedule(schedules []Schedule) string {
	if len(schedules) == 0 {
		return "Нет доступных слотов."
	}

	var builder strings.Builder
	builder.WriteString("Расписание:\n")
	for _, s := range schedules {
		builder.WriteString(fmt.Sprintf(
			"📅 %s - %s [%s]\n",
			formatTime(s.StartTime),
			formatTime(s.EndTime),
			s.Status,
		))
	}
	return builder.String()
}

// Генерация текста для отображения записей ученика
func FormatBookings(bookings []Schedule) string {
	if len(bookings) == 0 {
		return "У вас нет активных записей."
	}

	var builder strings.Builder
	builder.WriteString("Ваши записи:\n")
	for _, b := range bookings {
		builder.WriteString(fmt.Sprintf(
			"📌 %s - %s (%s)\n",
			formatTime(b.StartTime),
			formatTime(b.EndTime),
			b.Direction,
		))
	}
	return builder.String()
}

// Проверка валидности времени
func IsValidTime(timeStr string) bool {
	_, err := time.Parse("2006-01-02T15:04", timeStr)
	return err == nil
}

// Проверка, что время начала раньше времени окончания
func IsTimeRangeValid(start, end string) bool {
	startTime, err1 := time.Parse(time.RFC3339, start)
	endTime, err2 := time.Parse(time.RFC3339, end)
	if err1 != nil || err2 != nil {
		return false
	}
	return startTime.Before(endTime)
}

// Получение текущей недели в формате "ГГГГ-неделя"
func GetCurrentWeek() string {
	now := time.Now()
	year, week := now.ISOWeek()
	return fmt.Sprintf("%d-%02d", year, week)
}

// Генерация уникального идентификатора
func GenerateUniqueID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

// Проверка, что строка содержит только буквы и цифры
func IsAlphanumeric(s string) bool {
	for _, r := range s {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')) {
			return false
		}
	}
	return true
}

// Удаление лишних пробелов и форматирование текста
func CleanText(text string) string {
	return strings.Join(strings.Fields(text), " ")
}

// Проверка, что строка является допустимым направлением
func IdsValidDirection(direction string) bool {
	validDirections := []string{"ОГЭ", "ЕГЭ", "Путешествия", "Дети"}
	for _, d := range validDirections {
		if d == direction {
			return true
		}
	}
	return false
}

func isValidTime(timeStr string) bool {
	_, err := time.Parse("2006-01-02T15:04", timeStr)
	return err == nil
}
