package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

// Обработка команды /start
func handleStart(msg *tgbotapi.Message) {
	const teacherID = тут // Ваш Telegram ID

	// Удаляем предыдущее сообщение бота, если оно есть
	if lastID, exists := lastMessageID[msg.Chat.ID]; exists {
		deleteMessage(msg.Chat.ID, lastID)
	}

	// Проверяем, зарегистрирован ли пользователь
	exists, err := userExists(msg.Chat.ID)
	if err != nil {
		sendMessage(msg.Chat.ID, "Произошла ошибка. Попробуйте позже.")
		return
	}

	if !exists {
		var role string
		if msg.Chat.ID == teacherID {
			role = "teacher"
		} else {
			role = "student"
		}
		err := registerUser(msg.Chat.ID, role, msg.From.UserName)
		if err != nil {
			sendMessage(msg.Chat.ID, "Ошибка регистрации. Попробуйте снова.")
			return
		}
	}

	// Отправляем меню в зависимости от роли
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

// Меню для учителя
func showTeacherMenu(chatID int64) {
	// Получаем количество непрочитанных уведомлений
	unreadCount, err := countUnreadNotifications(chatID)
	if err != nil {
		unreadCount = 0
	}

	text := fmt.Sprintf("👨‍🏫 *Меню учителя*\nВыберите действие:\n📬 Уведомления: %d", unreadCount)
	buttons := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📅 Управление расписанием", "teacher_schedule"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("👥 Просмотр учеников", "teacher_students"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("➕ Добавить слот", "add_slot"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("📬 Уведомления (%d)", unreadCount), "notifications"),
		),
	)
	sendMessageWithKeyboard(chatID, text, &buttons)
}

// Меню для ученика
func showStudentMenu(chatID int64) {
	text := "👨‍🎓 *Меню ученика*\nВыберите действие:"
	buttons := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📅 Записаться на занятие", "student_book"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🗓 Мои записи", "student_bookings"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("❌ Отменить запись", "student_cancel"),
		),
	)
	sendMessageWithKeyboard(chatID, text, &buttons)
}

func sendTemporaryNotification(teacherID int64, message string) {
	err := addNotification(teacherID, message)
	if err != nil {
		fmt.Println("Ошибка добавления уведомления:", err)
		return
	}

	msg := tgbotapi.NewMessage(teacherID, message) // Отправляем только уведомление без текста "У вас новое уведомление"
	newMsg, err := bot.Send(msg)
	if err != nil {
		fmt.Println("Ошибка отправки уведомления:", err)
		return
	}

	// Удаляем сообщение через 5 секунд без вызова меню
	go func(chatID int64, msgID int) {
		time.Sleep(5 * time.Second)
		deleteMessage(chatID, msgID)
	}(teacherID, newMsg.MessageID)
}

func handleCallback(query *tgbotapi.CallbackQuery) {
	chatID := query.Message.Chat.ID
	messageID := query.Message.MessageID
	data := query.Data

	user, err := getUser(chatID)
	if err != nil {
		sendMessage(chatID, "Ошибка получения данных пользователя.")
		return
	}

	if user.Role == "teacher" && data != "notifications" && data != "delete_schedule" && !strings.HasPrefix(data, "mark_read_") && data != "clear_notifications" {
		deleteMessage(chatID, messageID)
	}

	switch data {
	case "back_to_menu":
		if user.Role == "teacher" {
			showTeacherMenu(chatID)
		} else {
			showStudentMenu(chatID)
		}
	case "teacher_schedule":
		go handleTeacherSchedule(chatID)
	case "teacher_students":
		go handleTeacherStudents(chatID)
	case "add_slot":
		fmt.Println("Handling 'add_slot' for chatID:", chatID) // Отладка
		showMonthCalendar(chatID, time.Now().Year(), time.Now().Month())
	case "student_book":
		showStudentMonthCalendar(chatID, time.Now().Year(), time.Now().Month())
	case "student_bookings":
		go handleStudentBookings(chatID)
	case "student_cancel":
		go handleStudentCancel(chatID)
	case "back_to_calendar":
		showMonthCalendar(chatID, time.Now().Year(), time.Now().Month())
	case "delete_schedule":
		handleDeleteSchedule(chatID, messageID)
	case "notifications":
		go func() {
			notifications, err := getTeacherNotifications(chatID)
			if err != nil {
				sendMessage(chatID, "Ошибка получения уведомлений.")
				return
			}
			if len(notifications) == 0 {
				buttons := tgbotapi.NewInlineKeyboardMarkup(
					tgbotapi.NewInlineKeyboardRow(
						tgbotapi.NewInlineKeyboardButtonData("↩️ Назад в меню", "back_to_menu"),
					),
				)
				sendMessageWithKeyboard(chatID, "📬 У вас нет уведомлений.", &buttons)
				return
			}
			var builder strings.Builder
			builder.WriteString("📬 *Ваши уведомления:*\n")
			var buttons [][]tgbotapi.InlineKeyboardButton
			for _, n := range notifications {
				status := "🔔"
				if n.IsRead {
					status = "✅"
				}
				builder.WriteString(fmt.Sprintf("%s %s\n", status, n.Message))
				if !n.IsRead {
					buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(
						tgbotapi.NewInlineKeyboardButtonData(
							fmt.Sprintf("Отметить прочитанным: %s", formatTime(n.CreatedAt)),
							fmt.Sprintf("mark_read_%d", n.ID),
						),
					))
				}
			}
			buttons = append(buttons,
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("🗑️ Очистить все", "clear_notifications"),
					tgbotapi.NewInlineKeyboardButtonData("↩️ Назад в меню", "back_to_menu"),
				),
			)
			keyboard := tgbotapi.NewInlineKeyboardMarkup(buttons...)
			sendMessageWithKeyboard(chatID, builder.String(), &keyboard)
		}()
	default:
		if data == "ignore" {
			return
		}
		if strings.HasPrefix(data, "confirm_reminder_") {
			scheduleIDStr := strings.TrimPrefix(data, "confirm_reminder_")
			scheduleID, err := strconv.ParseInt(scheduleIDStr, 10, 64)
			if err != nil {
				sendMessage(chatID, "Ошибка обработки подтверждения.")
				fmt.Println("Ошибка парсинга scheduleID:", err, "scheduleIDStr:", scheduleIDStr)
				return
			}

			fmt.Println("Confirmed reminder for schedule ID:", scheduleID, "chatID:", chatID)
			if lastID, exists := lastMessageID[chatID]; exists {
				deleteMessage(chatID, lastID)
				fmt.Println("Deleted previous message ID:", lastID)
			} else {
				fmt.Println("No previous message found in lastMessageID for chatID:", chatID)
			}
			showStudentMenu(chatID)
			return
		}

		if strings.HasPrefix(data, "add_slot_") {
			slotStr := strings.TrimPrefix(data, "add_slot_")
			go handleAddSlot(chatID, slotStr)
		} else if strings.HasPrefix(data, "calendar_") {
			dateStr := strings.TrimPrefix(data, "calendar_")
			fmt.Println("Teacher selected date:", dateStr)
			if strings.HasPrefix(dateStr, "prev_") || strings.HasPrefix(dateStr, "next_") {
				year, month, err := parseYearMonth(strings.TrimPrefix(dateStr, "prev_"))
				if err != nil {
					year, month, err = parseYearMonth(strings.TrimPrefix(dateStr, "next_"))
				}
				if err == nil {
					showMonthCalendar(chatID, year, month)
					return
				}
			}
			date, err := time.Parse("2006-01-02", dateStr)
			if err != nil {
				fmt.Println("Ошибка парсинга даты в handleCallback:", err)
				return
			}
			if date.Before(time.Now().Truncate(24 * time.Hour)) {
				return
			}
			showTimeSlots(chatID, dateStr)
		} else if strings.HasPrefix(data, "student_calendar_") {
			dateStr := strings.TrimPrefix(data, "student_calendar_")
			fmt.Println("Student selected date:", dateStr)
			if strings.HasPrefix(dateStr, "prev_") || strings.HasPrefix(dateStr, "next_") {
				year, month, err := parseYearMonth(strings.TrimPrefix(dateStr, "prev_"))
				if err != nil {
					year, month, err = parseYearMonth(strings.TrimPrefix(dateStr, "next_"))
				}
				if err == nil {
					showStudentMonthCalendar(chatID, year, month)
					return
				}
			}
			date, err := time.Parse("2006-01-02", dateStr)
			if err != nil {
				fmt.Println("Ошибка парсинга даты в handleCallback:", err)
				return
			}
			if date.Before(time.Now().Truncate(24 * time.Hour)) {
				return
			}
			showStudentTimeSlots(chatID, dateStr)
		} else if strings.HasPrefix(data, "book_") {
			slotIDStr := strings.TrimPrefix(data, "book_")
			slotID, err := strconv.ParseInt(slotIDStr, 10, 64)
			if err != nil {
				sendMessage(chatID, "Ошибка: неверный ID слота.")
			} else {
				go handleBooking(chatID, slotID)
			}
		} else if strings.HasPrefix(data, "cancel_") {
			slotIDStr := strings.TrimPrefix(data, "cancel_")
			slotID, err := strconv.ParseInt(slotIDStr, 10, 64)
			if err != nil {
				sendMessage(chatID, "Ошибка: неверный ID слота.")
			} else {
				go handleCancelBooking(chatID, slotID)
			}
		} else if strings.HasPrefix(data, "select_delete_") {
			slotIDStr := strings.TrimPrefix(data, "select_delete_")
			slotID, err := strconv.ParseInt(slotIDStr, 10, 64)
			if err != nil {
				sendMessage(chatID, "Ошибка: неверный ID слота.")
				return
			}
			handleSelectDeleteSlot(chatID, messageID, slotID)
		} else if strings.HasPrefix(data, "delete_slot_") {
			slotIDStr := strings.TrimPrefix(data, "delete_slot_")
			slotID, err := strconv.ParseInt(slotIDStr, 10, 64)
			if err != nil {
				sendMessage(chatID, "Ошибка: неверный ID слота.")
				return
			}
			err = DeleteScheduleSlot(slotID)
			if err != nil {
				sendMessage(chatID, "Ошибка удаления слота.")
				return
			}
			buttons := tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("📅 Вернуться к календарю", "back_to_calendar"),
					tgbotapi.NewInlineKeyboardButtonData("↩️ Вернуться в меню", "back_to_menu"),
				),
			)
			sendMessageWithKeyboard(chatID, "✅ Слот успешно удален", &buttons)
		} else if strings.HasPrefix(data, "mark_read_") {
			notificationIDStr := strings.TrimPrefix(data, "mark_read_")
			notificationID, err := strconv.Atoi(notificationIDStr)
			if err != nil {
				sendMessage(chatID, "Ошибка: неверный ID уведомления.")
				return
			}
			err = markNotificationAsRead(notificationID)
			if err != nil {
				sendMessage(chatID, "Ошибка отметки уведомления как прочитанного.")
				return
			}
			notifications, err := getTeacherNotifications(chatID)
			if err != nil {
				sendMessage(chatID, "Ошибка получения уведомлений.")
				return
			}
			var builder strings.Builder
			builder.WriteString("📬 *Ваши уведомления:*\n")
			var buttons [][]tgbotapi.InlineKeyboardButton
			for _, n := range notifications {
				status := "🔔"
				if n.IsRead {
					status = "✅"
				}
				builder.WriteString(fmt.Sprintf("%s %s\n", status, n.Message))
				if !n.IsRead {
					buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(
						tgbotapi.NewInlineKeyboardButtonData(
							fmt.Sprintf("Отметить прочитанным: %s", formatTime(n.CreatedAt)),
							fmt.Sprintf("mark_read_%d", n.ID),
						),
					))
				}
			}
			buttons = append(buttons,
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("🗑️ Очистить все", "clear_notifications"),
					tgbotapi.NewInlineKeyboardButtonData("↩️ Назад в меню", "back_to_menu"),
				),
			)
			keyboard := tgbotapi.NewInlineKeyboardMarkup(buttons...)
			sendMessageWithKeyboard(chatID, builder.String(), &keyboard)
		} else if data == "clear_notifications" {
			err := clearTeacherNotifications(chatID)
			if err != nil {
				sendMessage(chatID, "Ошибка очистки уведомлений.")
				return
			}
			buttons := tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("↩️ Назад в меню", "back_to_menu"),
				),
			)
			sendMessageWithKeyboard(chatID, "📬 Уведомления очищены.", &buttons)
		} else {
			if user.Role == "teacher" {
				showTeacherMenu(chatID)
			} else {
				showStudentMenu(chatID)
			}
		}
	}

	if _, err := bot.AnswerCallbackQuery(tgbotapi.NewCallback(query.ID, "")); err != nil {
		fmt.Println("Ошибка ответа на callback:", err)
	}
}

func handleSelectDeleteSlot(chatID int64, messageID int, slotID int64) {
	err := DeleteScheduleSlot(slotID)
	if err != nil {
		editMessage(chatID, messageID, "❌ Ошибка удаления слота")
		return
	}

	// Уведомляем об успешном удалении
	buttons := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("↩️ Назад к меню", "back_to_menu"),
		),
	)
	editMessageWithKeyboard(chatID, messageID, "✅ Слот успешно удален", &buttons)

	// Возвращаем пользователя в главное меню
	showTeacherMenu(chatID)
}

func handleDeleteSchedule(chatID int64, messageID int) {
	schedules, err := getTeacherSchedule(chatID)
	if err != nil {
		editMessage(chatID, messageID, "❌ Ошибка загрузки расписания")
		return
	}

	if len(schedules) == 0 {
		editMessage(chatID, messageID, "📭 Расписание пусто")
		return
	}

	var buttons [][]tgbotapi.InlineKeyboardButton
	for _, s := range schedules {
		buttonText := fmt.Sprintf("%s - %s", formatTime(s.StartTime), formatTime(s.EndTime))
		buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(buttonText, fmt.Sprintf("select_delete_%d", s.ID)),
		))
	}
	buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("↩️ Назад к меню", "back_to_menu"),
	))

	keyboard := tgbotapi.NewInlineKeyboardMarkup(buttons...)
	editMessageWithKeyboard(chatID, messageID, "Выберите слот для удаления:", &keyboard)
}

func editMessage(chatID int64, messageID int, text string) {
	edit := tgbotapi.NewEditMessageText(chatID, messageID, text)
	edit.ParseMode = "Markdown"
	if _, err := bot.Send(edit); err != nil {
	}
}

func editMessageWithKeyboard(chatID int64, messageID int, text string, keyboard *tgbotapi.InlineKeyboardMarkup) {
	editText := tgbotapi.NewEditMessageText(chatID, messageID, text)
	editText.ParseMode = "Markdown"
	if _, err := bot.Send(editText); err != nil {
		return
	}

	if keyboard != nil {
		editMarkup := tgbotapi.NewEditMessageReplyMarkup(chatID, messageID, *keyboard)
		if _, err := bot.Send(editMarkup); err != nil {
		}
	}
}

func showStudentMonthCalendar(chatID int64, year int, month time.Month) {
	startOfMonth := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
	endOfMonth := startOfMonth.AddDate(0, 1, -1)

	firstDay := startOfMonth.Weekday()
	firstDayOffset := (int(firstDay) - 1) % 7
	if firstDayOffset < 0 {
		firstDayOffset += 7
	}
	day := 1 - firstDayOffset

	var buttons [][]tgbotapi.InlineKeyboardButton

	navRow := []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData("<<", fmt.Sprintf("student_calendar_prev_%d-%02d", year, int(month)-1)),
		tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("%s %d", month.String(), year), "ignore"),
		tgbotapi.NewInlineKeyboardButtonData(">>", fmt.Sprintf("student_calendar_next_%d-%02d", year, int(month)+1)),
	}
	buttons = append(buttons, navRow)

	header := []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData("Пн", "ignore"),
		tgbotapi.NewInlineKeyboardButtonData("Вт", "ignore"),
		tgbotapi.NewInlineKeyboardButtonData("Ср", "ignore"),
		tgbotapi.NewInlineKeyboardButtonData("Чт", "ignore"),
		tgbotapi.NewInlineKeyboardButtonData("Пт", "ignore"),
		tgbotapi.NewInlineKeyboardButtonData("Сб", "ignore"),
		tgbotapi.NewInlineKeyboardButtonData("Вс", "ignore"),
	}
	buttons = append(buttons, header)

	for week := 0; week < 6; week++ {
		var weekRow []tgbotapi.InlineKeyboardButton
		for d := 0; d < 7; d++ {
			currentDay := day + d + week*7
			if currentDay >= 1 && currentDay <= endOfMonth.Day() {
				date := time.Date(year, month, currentDay, 0, 0, 0, 0, time.UTC)
				dateStr := date.Format("2006-01-02")
				buttonText := fmt.Sprintf("%2d", currentDay)
				var callbackData string

				if date.Before(time.Now().Truncate(24 * time.Hour)) {
					// Прошедшие дни некликабельны
					callbackData = "ignore"
				} else {
					// Текущие и будущие дни кликабельны
					callbackData = fmt.Sprintf("student_calendar_%s", dateStr)
				}

				button := tgbotapi.NewInlineKeyboardButtonData(buttonText, callbackData)
				weekRow = append(weekRow, button)
			} else {
				weekRow = append(weekRow, tgbotapi.NewInlineKeyboardButtonData("  ", "ignore"))
			}
		}
		buttons = append(buttons, weekRow)
		if day+7*(week+1) > endOfMonth.Day() {
			break
		}
	}

	buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("↩️ Назад к меню", "back_to_menu"),
	))

	keyboard := tgbotapi.NewInlineKeyboardMarkup(buttons...)
	sendMessageWithKeyboard(chatID, "Выберите дату для записи на занятие:", &keyboard)
}

func parseYearMonth(dateStr string) (int, time.Month, error) {
	// Проверяем, содержит ли строка только год и месяц
	if len(dateStr) != 7 || dateStr[4] != '-' {
		return 0, 0, fmt.Errorf("invalid date format: %s", dateStr)
	}
	year, err := strconv.Atoi(dateStr[:4])
	if err != nil {
		return 0, 0, err
	}
	month, err := strconv.Atoi(dateStr[5:])
	if err != nil {
		return 0, 0, err
	}
	if month < 1 || month > 12 {
		return 0, 0, fmt.Errorf("invalid month: %d", month)
	}
	return year, time.Month(month), nil
}

func showMonthCalendar(chatID int64, year int, month time.Month) {
	fmt.Println("showMonthCalendar called for chatID:", chatID, "year:", year, "month:", month) // Отладка

	startOfMonth := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
	endOfMonth := startOfMonth.AddDate(0, 1, -1)

	firstDay := startOfMonth.Weekday()
	firstDayOffset := (int(firstDay) - 1) % 7
	if firstDayOffset < 0 {
		firstDayOffset += 7
	}
	day := 1 - firstDayOffset

	var buttons [][]tgbotapi.InlineKeyboardButton

	navRow := []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData("<<", fmt.Sprintf("calendar_prev_%d-%02d", year, int(month)-1)),
		tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("%s %d", month.String(), year), "ignore"),
		tgbotapi.NewInlineKeyboardButtonData(">>", fmt.Sprintf("calendar_next_%d-%02d", year, int(month)+1)),
	}
	buttons = append(buttons, navRow)

	header := []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData("Пн", "ignore"),
		tgbotapi.NewInlineKeyboardButtonData("Вт", "ignore"),
		tgbotapi.NewInlineKeyboardButtonData("Ср", "ignore"),
		tgbotapi.NewInlineKeyboardButtonData("Чт", "ignore"),
		tgbotapi.NewInlineKeyboardButtonData("Пт", "ignore"),
		tgbotapi.NewInlineKeyboardButtonData("Сб", "ignore"),
		tgbotapi.NewInlineKeyboardButtonData("Вс", "ignore"),
	}
	buttons = append(buttons, header)

	for week := 0; week < 6; week++ {
		var weekRow []tgbotapi.InlineKeyboardButton
		for d := 0; d < 7; d++ {
			currentDay := day + d + week*7
			if currentDay >= 1 && currentDay <= endOfMonth.Day() {
				date := time.Date(year, month, currentDay, 0, 0, 0, 0, time.UTC)
				dateStr := date.Format("2006-01-02")
				buttonText := fmt.Sprintf("%2d", currentDay)
				var callbackData string

				if date.Before(time.Now().Truncate(24 * time.Hour)) {
					callbackData = "ignore"
				} else {
					callbackData = fmt.Sprintf("calendar_%s", dateStr)
				}

				button := tgbotapi.NewInlineKeyboardButtonData(buttonText, callbackData)
				weekRow = append(weekRow, button)
			} else {
				weekRow = append(weekRow, tgbotapi.NewInlineKeyboardButtonData("  ", "ignore"))
			}
		}
		buttons = append(buttons, weekRow)
		if day+7*(week+1) > endOfMonth.Day() {
			break
		}
	}

	buttons = append(buttons, []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData("↩️ Назад к меню", "back_to_menu"),
	})

	keyboard := tgbotapi.NewInlineKeyboardMarkup(buttons...)
	msg := tgbotapi.NewMessage(chatID, "Выберите дату для добавления слота:")
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard

	// Упрощаем отправку: всегда отправляем новое сообщение, удаляя старое
	if lastID, exists := lastMessageID[chatID]; exists {
		deleteMessage(chatID, lastID)
		fmt.Println("Deleted previous message ID:", lastID) // Отладка
	}
	newMsg, err := bot.Send(msg)
	if err != nil {
		fmt.Println("Ошибка отправки календаря:", err) // Отладка
		return
	}
	lastMessageID[chatID] = newMsg.MessageID
	fmt.Println("Calendar sent, new message ID:", newMsg.MessageID) // Отладка
}

// Новая функция для показа календаря
func showCalendar(chatID int64) {
	now := time.Now()
	currentYear, currentMonth, _ := now.Date()
	startOfMonth := time.Date(currentYear, currentMonth, 1, 0, 0, 0, 0, time.UTC)
	endOfMonth := startOfMonth.AddDate(0, 1, -1)

	// Определяем первый день недели (например, понедельник)
	firstDay := startOfMonth.Weekday()
	if firstDay == time.Sunday {
		firstDay = 0 // Считаем понедельник как 0 для удобства
	} else {
		firstDay-- // Сдвигаем, чтобы понедельник был 0
	}

	var buttons [][]tgbotapi.InlineKeyboardButton
	var weekRow []tgbotapi.InlineKeyboardButton
	day := 1 - int(firstDay) // Начинаем с дня до начала месяца, чтобы заполнить пустые ячейки

	for row := 0; row < 6; row++ { // Максимум 6 недель
		weekRow = nil
		for col := 0; col < 7; col++ { // 7 дней недели
			if day >= 1 && day <= endOfMonth.Day() {
				date := time.Date(currentYear, currentMonth, day, 0, 0, 0, 0, time.UTC)
				dateStr := date.Format("2006-01-02")
				color := "⬜" // Белый квадрат по умолчанию

				// Проверяем, текущий ли это день
				if date.Equal(now) {
					color = "🟩" // Зелёный для текущего дня
				}

				// Проверяем, есть ли доступные слоты для этой даты
				slots, err := getSlotsForDate(date)
				if err == nil && len(slots) > 0 {
					hasFreeSlots := false
					for _, slot := range slots {
						if slot.Status == "free" {
							hasFreeSlots = true
							break
						}
					}
					if !hasFreeSlots {
						color = "🟥" // Красный для дней без свободных слотов
					}
				}

				button := tgbotapi.NewInlineKeyboardButtonData(
					fmt.Sprintf("%s %d", color, day),
					fmt.Sprintf("calendar_%s", dateStr),
				)
				weekRow = append(weekRow, button)
				day++
			} else {
				// Пустые ячейки для начала или конца месяца
				weekRow = append(weekRow, tgbotapi.NewInlineKeyboardButtonData("⬜", "ignore"))
			}
		}
		if len(weekRow) > 0 {
			buttons = append(buttons, weekRow)
		}
		if day > endOfMonth.Day() {
			break
		}
	}

	// Добавляем кнопку "Следующий месяц" для навигации
	if currentMonth < time.December {
		nextMonth := time.Date(currentYear, currentMonth+1, 1, 0, 0, 0, 0, time.UTC)
		nextMonthStr := nextMonth.Format("2006-01")
		buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("➡️ Следующий месяц", fmt.Sprintf("calendar_next_%s", nextMonthStr)),
		))
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(buttons...)
	sendMessageWithKeyboard(chatID, fmt.Sprintf("Календарь %s %d для добавления слота:", currentMonth, currentYear), &keyboard)
}

// Новая функция для показа временных слотов
func showTimeSlots(chatID int64, dateStr string) {
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		sendMessage(chatID, "Ошибка обработки даты: неверный формат.")
		fmt.Println("Ошибка парсинга даты:", err, "dateStr:", dateStr)
		return
	}

	// Если дата прошедшая, просто игнорируем нажатие
	if date.Before(time.Now().Truncate(24 * time.Hour)) {
		return
	}

	slots, err := getSlotsForDate(date)
	if err != nil {
		sendMessage(chatID, "Ошибка получения слотов.")
		fmt.Println("Ошибка getSlotsForDate:", err)
		return
	}

	var buttons [][]tgbotapi.InlineKeyboardButton
	var row []tgbotapi.InlineKeyboardButton

	for hour := 9; hour < 22; hour++ {
		startTime := time.Date(date.Year(), date.Month(), date.Day(), hour, 0, 0, 0, time.UTC)
		slotExists := false
		for _, slot := range slots {
			existingStart, err := time.Parse(time.RFC3339, slot.StartTime)
			if err == nil && existingStart.Equal(startTime) {
				slotExists = true
				break
			}
		}

		color := "⬜"                       // Белый по умолчанию
		if !startTime.Before(time.Now()) { // Только текущее и будущее время
			if slotExists {
				color = "🟥" // Красный для занятого времени
			} else {
				color = "🟩" // Зеленый для свободного времени
			}
		}

		timeKey := fmt.Sprintf("%s_%02d:00", dateStr, hour)
		btn := tgbotapi.NewInlineKeyboardButtonData(
			fmt.Sprintf("%02d:00\n%s", hour, color), // Часы над цветом
			fmt.Sprintf("add_slot_%s", timeKey),
		)
		row = append(row, btn)
		if len(row) == 4 {
			buttons = append(buttons, row)
			row = []tgbotapi.InlineKeyboardButton{}
		}
	}

	if len(row) > 0 {
		buttons = append(buttons, row)
	}

	buttons = append(buttons, []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData("↩️ Назад к календарю", "add_slot"),
		tgbotapi.NewInlineKeyboardButtonData("↩️ Назад к меню", "back_to_menu"),
	})

	keyboard := tgbotapi.NewInlineKeyboardMarkup(buttons...)
	if lastID, exists := lastMessageID[chatID]; exists {
		deleteMessage(chatID, lastID)
	}
	msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("🕒 *Доступные часы на %s:*\n\n🟩 - Свободно\n🟥 - Занято", dateStr))
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard
	newMsg, err := bot.Send(msg)
	if err != nil {
		fmt.Println("Ошибка отправки временных слотов:", err)
		return
	}
	lastMessageID[chatID] = newMsg.MessageID
}

// Новая функция для добавления слота
func handleAddSlot(chatID int64, slotStr string) {
	parts := strings.Split(slotStr, "_")
	if len(parts) != 2 {
		sendMessage(chatID, "Ошибка: неверный формат слота.")
		return
	}
	dateStr := parts[0]
	timeStr := parts[1]

	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		sendMessage(chatID, "Ошибка: неверная дата.")
		return
	}
	timeParts := strings.Split(timeStr, ":")
	if len(timeParts) != 2 {
		sendMessage(chatID, "Ошибка: неверное время.")
		return
	}
	hour, err := strconv.Atoi(timeParts[0])
	if err != nil || hour < 0 || hour > 23 {
		sendMessage(chatID, "Ошибка: неверный час.")
		return
	}
	minute, err := strconv.Atoi(timeParts[1])
	if err != nil || minute < 0 || minute > 59 {
		sendMessage(chatID, "Ошибка: неверная минута.")
		return
	}

	startTime := time.Date(date.Year(), date.Month(), date.Day(), hour, minute, 0, 0, time.UTC)
	if startTime.Before(time.Now()) {
		sendMessage(chatID, "Нельзя добавить слот на прошедшее время.")
		return
	}

	endTime := startTime.Add(1 * time.Hour)
	startTimeStr := startTime.Format(time.RFC3339)
	endTimeStr := endTime.Format(time.RFC3339)

	var slotID int64
	err = db.QueryRow(
		`SELECT id FROM schedules WHERE teacher_id = ? AND start_time = ?`,
		chatID, startTimeStr).Scan(&slotID)

	if err == nil {
		buttons := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🗑️ Удалить слот", fmt.Sprintf("delete_slot_%d", slotID)),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("📅 Вернуться к календарю", "back_to_calendar"),
				tgbotapi.NewInlineKeyboardButtonData("↩️ Вернуться в меню", "back_to_menu"),
			),
		)
		sendMessageWithKeyboard(chatID, fmt.Sprintf("Слот %s - %s уже занят. Что делать?", startTime.Format("15:04"), endTime.Format("15:04")), &buttons)
		return
	}

	err = addScheduleSlot(chatID, startTimeStr, endTimeStr)
	if err != nil {
		sendMessage(chatID, "Ошибка добавления слота.")
		return
	}

	buttons := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📅 Вернуться к календарю", "back_to_calendar"),
			tgbotapi.NewInlineKeyboardButtonData("↩️ Вернуться в меню", "back_to_menu"),
		),
	)
	sendMessageWithKeyboard(chatID, fmt.Sprintf("Слот успешно добавлен: %s - %s", startTime.Format("15:04"), endTime.Format("15:04")), &buttons)
}

// Управление расписанием (для учителя)
func handleTeacherSchedule(chatID int64) {
	// Проверка прав учителя
	user, err := getUser(chatID)
	if err != nil || user.Role != "teacher" {
		sendMessage(chatID, "❌ Эта команда доступна только учителям")
		return
	}

	schedules, err := getTeacherSchedule(chatID)
	if err != nil {
		sendMessage(chatID, "❌ Ошибка загрузки расписания")
		return
	}

	if len(schedules) == 0 {
		sendMessage(chatID, "📭 Расписание пусто")
		return
	}

	var builder strings.Builder
	builder.WriteString("📅 *Ваше расписание:*\n\n")

	for _, s := range schedules {
		builder.WriteString(fmt.Sprintf(
			"⏰ %s - %s\n🔄 Статус: %s\n",
			formatTime(s.StartTime),
			formatTime(s.EndTime),
			statusToEmoji(s.Status),
		))
	}

	// Кнопки управления (только удаление и назад)
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🗑️ Удалить", "delete_schedule"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("↩️ Назад к меню", "back_to_menu"),
		),
	)

	sendMessageWithKeyboard(chatID, builder.String(), &keyboard)
}

func statusToEmoji(status string) string {
	if status == "booked" {
		return "✅ Занят"
	}
	return "🆓 Свободен"
}

// Просмотр учеников (для учителя)
func handleTeacherStudents(chatID int64) {
	students, err := getTeacherStudents(chatID)
	if err != nil {
		sendMessage(chatID, "Ошибка получения списка учеников.")
		return
	}

	if len(students) == 0 {
		// Создаем клавиатуру с кнопкой "Назад"
		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("↩️ Назад к меню", "back_to_menu"),
			),
		)
		sendMessageWithKeyboard(chatID, "У вас пока нет учеников.", &keyboard)
		return
	}

	var builder strings.Builder
	builder.WriteString("👥 *Ваши ученики:*\n")
	for _, s := range students {
		builder.WriteString(fmt.Sprintf(
			"👤 @%s - %s (%s)\n",
			s.StudentUsername,
			formatTime(s.StartTime),
			s.Direction,
		))
	}

	// Кнопка возврата для случая, когда ученики есть
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("↩️ Назад к меню", "back_to_menu"),
		),
	)

	sendMessageWithKeyboard(chatID, builder.String(), &keyboard)
}

// Запись на занятие (для ученика)
func handleStudentBook(chatID int64) {
	showStudentCalendar(chatID)
}

func showStudentCalendar(chatID int64) {
	now := time.Now()
	currentYear, currentMonth, _ := now.Date()
	startOfMonth := time.Date(currentYear, currentMonth, 1, 0, 0, 0, 0, time.UTC)
	endOfMonth := startOfMonth.AddDate(0, 1, -1)

	// Определяем первый день недели
	firstDay := startOfMonth.Weekday()
	if firstDay == time.Sunday {
		firstDay = 0 // Считаем понедельник как 0
	} else {
		firstDay-- // Сдвигаем, чтобы понедельник был 0
	}

	var buttons [][]tgbotapi.InlineKeyboardButton
	var weekRow []tgbotapi.InlineKeyboardButton
	day := 1 - int(firstDay) // Начинаем с дня до начала месяца

	for row := 0; row < 6; row++ { // Максимум 6 недель
		weekRow = nil
		for col := 0; col < 7; col++ { // 7 дней недели
			if day >= 1 && day <= endOfMonth.Day() {
				date := time.Date(currentYear, currentMonth, day, 0, 0, 0, 0, time.UTC)
				dateStr := date.Format("2006-01-02")
				color := "⬜" // Белый квадрат по умолчанию

				// Проверяем, текущий ли это день
				if date.Equal(now) {
					color = "🟩" // Зелёный для текущего дня
				}

				// Проверяем, есть ли доступные слоты для этой даты
				slots, err := getAvailableSlotsForDate(date)
				if err == nil && len(slots) > 0 {
					color = "🟩" // Зелёный для дней с доступными слотами
				} else if err != nil || len(slots) == 0 {
					color = "🟥" // Красный для дней без свободных слотов
				}

				button := tgbotapi.NewInlineKeyboardButtonData(
					fmt.Sprintf("%s %d", color, day),
					fmt.Sprintf("student_calendar_%s", dateStr),
				)
				weekRow = append(weekRow, button)
				day++
			} else {
				// Пустые ячейки для начала или конца месяца
				weekRow = append(weekRow, tgbotapi.NewInlineKeyboardButtonData("⬜", "ignore"))
			}
		}
		if len(weekRow) > 0 {
			buttons = append(buttons, weekRow)
		}
		if day > endOfMonth.Day() {
			break
		}
	}

	// Добавляем кнопки навигации
	var navRow []tgbotapi.InlineKeyboardButton
	if currentMonth > time.January {
		prevMonth := time.Date(currentYear, currentMonth-1, 1, 0, 0, 0, 0, time.UTC)
		prevMonthStr := prevMonth.Format("2006-01")
		navRow = append(navRow, tgbotapi.NewInlineKeyboardButtonData("⬅️ Предыдущий месяц", fmt.Sprintf("student_calendar_next_%s", prevMonthStr)))
	}
	if currentMonth < time.December {
		nextMonth := time.Date(currentYear, currentMonth+1, 1, 0, 0, 0, 0, time.UTC)
		nextMonthStr := nextMonth.Format("2006-01")
		navRow = append(navRow, tgbotapi.NewInlineKeyboardButtonData("➡️ Следующий месяц", fmt.Sprintf("student_calendar_next_%s", nextMonthStr)))
	}
	if len(navRow) > 0 {
		buttons = append(buttons, navRow)
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(buttons...)
	sendMessageWithKeyboard(chatID, fmt.Sprintf("Календарь %s %d для записи на занятие:", currentMonth, currentYear), &keyboard)
}

// Новая функция для показа слотов ученику
func showStudentTimeSlots(chatID int64, dateStr string) {
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		sendMessage(chatID, "Ошибка обработки даты.")
		return
	}

	slots, err := getAvailableSlotsForDate(date)
	if err != nil {
		sendMessage(chatID, "Ошибка получения слотов.")
		return
	}

	var buttons [][]tgbotapi.InlineKeyboardButton
	var row []tgbotapi.InlineKeyboardButton

	for _, slot := range slots {
		startTime, err := time.Parse(time.RFC3339, slot.StartTime)
		if err != nil {
			fmt.Println("Ошибка парсинга времени слота:", err)
			continue
		}
		color := "🟩" // Зеленый для свободных слотов
		if startTime.Before(time.Now()) {
			color = "🔴" // Красный для прошедшего времени
		}

		hour := startTime.Hour()
		btn := tgbotapi.NewInlineKeyboardButtonData(
			fmt.Sprintf("%02d:00\n%s", hour, color), // Часы над цветом
			fmt.Sprintf("book_%d", slot.ID),
		)
		row = append(row, btn)
		if len(row) == 3 {
			buttons = append(buttons, row)
			row = []tgbotapi.InlineKeyboardButton{}
		}
	}

	if len(row) > 0 {
		buttons = append(buttons, row)
	}

	navRow := []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData("↩️ К календарю", "student_book"),
		tgbotapi.NewInlineKeyboardButtonData("↩️ Назад к меню", "back_to_menu"),
	}
	buttons = append(buttons, navRow)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(buttons...)
	sendMessageWithKeyboard(chatID, fmt.Sprintf("✅ *Свободные слоты на %s:*\n\nВыберите удобное время:", dateStr), &keyboard)
}

// Новая функция для получения доступных слотов по дате
func getAvailableSlotsForDate(date time.Time) ([]Schedule, error) {
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC).Format(time.RFC3339)
	endOfDay := time.Date(date.Year(), date.Month(), date.Day(), 23, 59, 59, 0, time.UTC).Format(time.RFC3339)

	query := `SELECT id, start_time, end_time, status 
        FROM schedules 
        WHERE status = 'free' 
        AND start_time BETWEEN ? AND ? 
        ORDER BY start_time`
	rows, err := db.Query(query, startOfDay, endOfDay)
	if err != nil {
		return nil, fmt.Errorf("ошибка запроса свободных слотов: %v", err)
	}
	defer rows.Close()

	var slots []Schedule
	for rows.Next() {
		var s Schedule
		if err := rows.Scan(&s.ID, &s.StartTime, &s.EndTime, &s.Status); err != nil {
			return nil, fmt.Errorf("ошибка сканирования слотов: %v", err)
		}
		slots = append(slots, s)
	}
	return slots, nil
}

// Просмотр записей (для ученика)
func handleStudentBookings(chatID int64) {
	bookings, err := getStudentBookings(chatID)
	if err != nil {
		sendMessage(chatID, "Ошибка получения записей.")
		return
	}

	if len(bookings) == 0 {
		buttons := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("↩️ Назад в меню", "back_to_menu"),
			),
		)
		sendMessageWithKeyboard(chatID, "У вас нет активных записей.", &buttons)
		return
	}

	var builder strings.Builder
	builder.WriteString("🗓 *Ваши записи:*\n")
	for _, b := range bookings {
		builder.WriteString(fmt.Sprintf(
			"🕒 %s - %s (%s)\n",
			formatTime(b.StartTime),
			formatTime(b.EndTime),
			b.Direction.String,
		))
	}

	// Добавляем кнопку "Назад в меню"
	buttons := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("↩️ Назад в меню", "back_to_menu"),
		),
	)
	sendMessageWithKeyboard(chatID, builder.String(), &buttons)
}

// Отмена записи (для ученика)
func handleStudentCancel(chatID int64) {
	bookings, err := getStudentBookings(chatID)
	if err != nil {
		sendMessage(chatID, "Ошибка получения записей.")
		return
	}

	if len(bookings) == 0 {
		buttons := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("↩️ Назад в меню", "back_to_menu"),
			),
		)
		sendMessageWithKeyboard(chatID, "У вас нет активных записей.", &buttons)
		return
	}

	var buttons [][]tgbotapi.InlineKeyboardButton
	for _, b := range bookings {
		buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				fmt.Sprintf("🕒 %s", formatTime(b.StartTime)),
				fmt.Sprintf("cancel_%d", b.ID),
			),
		))
	}

	// Добавляем кнопку "Назад в меню"
	buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("↩️ Назад в меню", "back_to_menu"),
	))

	keyboard := tgbotapi.NewInlineKeyboardMarkup(buttons...)
	sendMessageWithKeyboard(chatID, "Выберите запись для отмены:", &keyboard)
}

func getUsername(chatID int64) string {
	var username string
	err := db.QueryRow("SELECT username FROM users WHERE telegram_id = ?", chatID).Scan(&username)
	if err != nil {
		// Если пользователь не найден или username пустой, возвращаем ID в качестве запасного варианта
		fmt.Println("Ошибка получения username:", err)
		return fmt.Sprintf("ID%d", chatID)
	}
	if username == "" {
		return fmt.Sprintf("ID%d", chatID)
	}
	return username
}

// Обработка бронирования слота
func handleBooking(chatID int64, slotID int64) {
	slot, err := getScheduleByID(slotID)
	if err != nil {
		sendMessage(chatID, "Ошибка: слот не найден.")
		return
	}

	if slot.Status != "free" {
		sendMessage(chatID, "Этот слот уже занят.")
		return
	}

	// Проверка на прошлые даты и время
	startTime, err := time.Parse(time.RFC3339, slot.StartTime)
	if err != nil {
		sendMessage(chatID, "Ошибка обработки времени слота.")
		return
	}
	if startTime.Before(time.Now()) {
		sendMessage(chatID, "Нельзя записаться на прошедшее время.")
		return
	}

	direction := "Общее"
	if slot.Direction.Valid {
		direction = slot.Direction.String
	}
	studentID := chatID

	err = updateScheduleStatus(slotID, "booked", studentID, direction)
	if err != nil {
		sendMessage(chatID, "Ошибка при записи на занятие.")
		return
	}

	buttons := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("↩️ Назад к меню", "back_to_menu"),
		),
	)
	sendMessageWithKeyboard(chatID, fmt.Sprintf("Вы успешно записаны на занятие: %s", formatTime(slot.StartTime)), &buttons)

	teacherID := slot.TeacherID
	teacherMsg := fmt.Sprintf("Новая запись:\n%s - %s\nУченик: @%s\nНаправление: %s",
		formatTime(slot.StartTime),
		formatTime(slot.EndTime),
		getUsername(chatID),
		direction)
	err = addNotification(teacherID, teacherMsg)
	if err != nil {
		fmt.Println("Ошибка добавления уведомления о записи:", err)
	}

	showTeacherMenu(teacherID)
}

// handlers.go
func handleCancelBooking(chatID int64, slotID int64) {
	slot, err := getScheduleByID(slotID)
	if err != nil {
		sendMessage(chatID, "Ошибка: слот не найден.")
		return
	}

	if !slot.StudentID.Valid || slot.StudentID.Int64 != chatID {
		sendMessage(chatID, "Вы не можете отменить чужую запись.")
		return
	}

	err = updateScheduleStatus(slotID, "free", 0, "")
	if err != nil {
		sendMessage(chatID, "Ошибка при отмене записи.")
		return
	}

	// Уведомляем ученика с кнопкой "Назад"
	buttons := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("↩️ Назад к меню", "back_to_menu"),
		),
	)
	sendMessageWithKeyboard(chatID, fmt.Sprintf("Запись на %s успешно отменена.", formatTime(slot.StartTime)), &buttons)

	// Уведомляем учителя через систему уведомлений и обновляем меню
	teacherID := slot.TeacherID
	teacherMsg := fmt.Sprintf("Ученик отменил занятие:\n%s - %s\nУченик: @%s",
		formatTime(slot.StartTime),
		formatTime(slot.EndTime),
		getUsername(chatID))
	err = addNotification(teacherID, teacherMsg)
	if err != nil {
		fmt.Println("Ошибка добавления уведомления об отмене:", err)
	}

	// Обновляем меню учителя
	showTeacherMenu(teacherID)
}
