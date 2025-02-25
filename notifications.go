package main

import (
	"fmt"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

const GroupChatID = -850434834 // Исправленный ID группы

// Запуск планировщика уведомлений
func StartNotificationScheduler() {
	go scheduleNotifier()
	go bookingNotifications()
	go cancellationNotifications()
	go lessonReminders()
}

// Еженедельное напоминание учителям о заполнении расписания
func WeeklyReminder() {
	for {
		now := time.Now()
		nextSunday := calculateNextSunday(now)
		sleepDuration := nextSunday.Sub(now)
		time.Sleep(sleepDuration)

		// Отправка уведомлений всем учителям
		teachers, err := getAllTeachers()
		if err != nil {
			continue
		}

		for _, teacher := range teachers {
			sendMessage(teacher.TelegramID, "Пора заполнить расписание на следующую неделю!")
		}
	}
}

// Уведомления о новых записях
func bookingNotifications() {
	// Лог удален
	for {
		time.Sleep(1 * time.Second)

		bookings, err := getNewBookings()
		if err != nil {
			// Лог удален
			continue
		}
		// Лог удален

		for _, booking := range bookings {
			teacherMsg := fmt.Sprintf("Новая запись:\n%s - %s\nУченик: @%s\nНаправление: %s",
				formatTime(booking.StartTime),
				formatTime(booking.EndTime),
				booking.StudentUsername,
				booking.Direction)
			sendTemporaryNotification(booking.TeacherID, teacherMsg)

			studentMsg := fmt.Sprintf("Вы записаны на занятие:\n%s - %s\nНаправление: %s",
				formatTime(booking.StartTime),
				formatTime(booking.EndTime),
				booking.Direction)
			keyboard := tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("↩️ Назад к меню", "back_to_menu"),
				),
			)
			sendMessageWithKeyboard(booking.StudentID, studentMsg, &keyboard)

			if err := markBookingAsNotified(booking.ID); err != nil {
				// Лог удален
			}
		}
	}
}

// Уведомления об отменах занятий
func cancellationNotifications() {
	// Лог удален
	for {
		time.Sleep(1 * time.Second)

		cancellations, err := getNewCancellations()
		if err != nil {
			// Лог удален
			continue
		}
		// Лог удален

		for _, cancel := range cancellations {
			teacherMsg := fmt.Sprintf("Запись отменена:\n%s - %s\nУченик: @%s",
				formatTime(cancel.StartTime),
				formatTime(cancel.EndTime),
				cancel.StudentUsername)
			sendTemporaryNotification(cancel.TeacherID, teacherMsg)

			studentMsg := fmt.Sprintf("Ваша запись отменена:\n%s - %s",
				formatTime(cancel.StartTime),
				formatTime(cancel.EndTime))
			keyboard := tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("↩️ Назад к меню", "back_to_menu"),
				),
			)
			sendMessageWithKeyboard(cancel.StudentID, studentMsg, &keyboard)

			if err := markCancellationAsNotified(cancel.ID); err != nil {
				// Лог удален
			}
		}
	}
}

func lessonReminders() {
	for {
		time.Sleep(1 * time.Minute)

		query := `SELECT id, teacher_id, student_id, start_time, end_time, direction, u.username 
                  FROM schedules s
                  JOIN users u ON s.student_id = u.telegram_id
                  WHERE s.status = 'booked' AND s.start_time > ?`
		rows, err := db.Query(query, time.Now().Format(time.RFC3339))
		if err != nil {
			fmt.Println("Ошибка запроса расписания:", err)
			continue
		}
		defer rows.Close()

		for rows.Next() {
			var b BookingNotification
			if err := rows.Scan(&b.ID, &b.TeacherID, &b.StudentID, &b.StartTime, &b.EndTime, &b.Direction, &b.StudentUsername); err != nil {
				fmt.Println("Ошибка сканирования записи:", err)
				continue
			}

			startTime, err := time.Parse(time.RFC3339, b.StartTime)
			if err != nil {
				fmt.Println("Ошибка парсинга времени:", err)
				continue
			}

			timeUntilStart := startTime.Sub(time.Now())

			// Уведомление для учителя и группы за 10 минут
			if timeUntilStart > 9*time.Minute && timeUntilStart <= 10*time.Minute {
				teacherMsg := fmt.Sprintf("Напоминание: урок через 10 минут!\n%s - %s\nУченик: @%s\nНаправление: %s",
					formatTime(b.StartTime),
					formatTime(b.EndTime),
					b.StudentUsername,
					b.Direction)
				sendTemporaryNotification(b.TeacherID, teacherMsg)

				channelMsg := fmt.Sprintf("Напоминание: урок начнется через 10 минут!\n%s - %s\nУченик: @%s\nНаправление: %s",
					formatTime(b.StartTime),
					formatTime(b.EndTime),
					b.StudentUsername,
					b.Direction)
				sendMessage(GroupChatID, channelMsg)
			}

			// Уведомление для ученика за 30 минут
			if timeUntilStart > 29*time.Minute && timeUntilStart <= 30*time.Minute {
				studentMsg := fmt.Sprintf("Напоминание: занятие через 30 минут!\n%s - %s\nНаправление: %s",
					formatTime(b.StartTime),
					formatTime(b.EndTime),
					b.Direction)

				keyboard := tgbotapi.NewInlineKeyboardMarkup(
					tgbotapi.NewInlineKeyboardRow(
						tgbotapi.NewInlineKeyboardButtonData("Хорошо, я уведомлен", fmt.Sprintf("confirm_reminder_%d", b.ID)),
					),
				)

				if lastID, exists := lastMessageID[b.StudentID]; exists {
					deleteMessage(b.StudentID, lastID)
				}
				msg := tgbotapi.NewMessage(b.StudentID, studentMsg)
				msg.ParseMode = "Markdown"
				msg.ReplyMarkup = keyboard
				newMsg, err := bot.Send(msg)
				if err != nil {
					fmt.Println("Ошибка отправки уведомления ученику:", err, "studentID:", b.StudentID)
					continue
				}
				lastMessageID[b.StudentID] = newMsg.MessageID
				fmt.Println("Sent reminder to studentID:", b.StudentID, "messageID:", newMsg.MessageID)
			}
		}
	}
}

// Вычисление следующего воскресенья
func calculateNextSunday(now time.Time) time.Time {
	daysUntilSunday := (7 - int(now.Weekday())) % 7
	nextSunday := now.AddDate(0, 0, daysUntilSunday)
	return time.Date(nextSunday.Year(), nextSunday.Month(), nextSunday.Day(), 18, 0, 0, 0, time.Local)
}

func scheduleNotifier() {
	for {
		now := time.Now()
		nextSunday := calculateNextSunday(now)
		sleepDuration := nextSunday.Sub(now)
		time.Sleep(sleepDuration)

		// Отправка уведомлений всем учителям
		teachers, err := getAllTeachers()
		if err != nil {
			continue
		}

		for _, teacher := range teachers {
			sendMessage(teacher.TelegramID, "Пора заполнить расписание на следующую неделю!")
		}
	}
}

// Форматирование времени
func formatTime(timeStr string) string {
	t, err := time.Parse(time.RFC3339, timeStr)
	if err != nil {
		return timeStr
	}
	return t.Format("02.01 15:04")
}

// Получение всех учителей
func getAllTeachers() ([]User, error) {
	query := `SELECT id, telegram_id, username FROM users WHERE role = 'teacher'`
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var teachers []User
	for rows.Next() {
		var t User
		if err := rows.Scan(&t.ID, &t.TelegramID, &t.Username); err != nil {
			return nil, err
		}
		teachers = append(teachers, t)
	}
	return teachers, nil
}

// Получение новых записей
func getNewBookings() ([]BookingNotification, error) {
	query := `SELECT s.id, s.teacher_id, s.student_id, s.start_time, s.end_time, s.direction, u.username 
		FROM schedules s
		JOIN users u ON s.student_id = u.telegram_id
		WHERE s.status = 'booked' AND s.notified = 0`
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var bookings []BookingNotification
	for rows.Next() {
		var b BookingNotification
		if err := rows.Scan(&b.ID, &b.TeacherID, &b.StudentID, &b.StartTime, &b.EndTime, &b.Direction, &b.StudentUsername); err != nil {
			return nil, err
		}
		bookings = append(bookings, b)
	}
	return bookings, nil
}

// Получение новых отмен
func getNewCancellations() ([]CancellationNotification, error) {
	query := `SELECT s.id, s.teacher_id, s.student_id, s.start_time, s.end_time, u.username 
              FROM schedules s
              JOIN users u ON s.student_id = u.telegram_id
              WHERE s.status = 'free' AND s.notified = 1`
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cancellations []CancellationNotification
	for rows.Next() {
		var c CancellationNotification
		if err := rows.Scan(&c.ID, &c.TeacherID, &c.StudentID, &c.StartTime, &c.EndTime, &c.StudentUsername); err != nil {
			return nil, err
		}
		cancellations = append(cancellations, c)
	}
	return cancellations, nil
}

// Пометить запись как уведомленную
func markBookingAsNotified(bookingID int) error {
	query := `UPDATE schedules SET notified = 1 WHERE id = ?`
	_, err := db.Exec(query, bookingID)
	return err
}

// Пометить отмену как уведомленную
func markCancellationAsNotified(cancellationID int) error {
	query := `UPDATE schedules SET notified = 0 WHERE id = ?`
	_, err := db.Exec(query, cancellationID)
	return err
}

func addNotification(teacherID int64, message string) error {
	query := `INSERT INTO notifications (teacher_id, message, created_at) VALUES (?, ?, ?)`
	_, err := db.Exec(query, teacherID, message, time.Now().Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("ошибка добавления уведомления: %v", err)
	}
	return nil
}

// Получение всех уведомлений для учителя
func getTeacherNotifications(chatID int64) ([]Notification, error) {
	rows, err := db.Query("SELECT id, message, is_read, created_at FROM notifications WHERE teacher_id = ? ORDER BY created_at DESC", chatID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notifications []Notification
	for rows.Next() {
		var n Notification
		if err := rows.Scan(&n.ID, &n.Message, &n.IsRead, &n.CreatedAt); err != nil {
			return nil, err
		}
		notifications = append(notifications, n)
	}
	return notifications, nil
}

// Подсчет непрочитанных уведомлений
func countUnreadNotifications(teacherID int64) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM notifications WHERE teacher_id = ? AND is_read = 0`
	err := db.QueryRow(query, teacherID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("ошибка подсчета уведомлений: %v", err)
	}
	return count, nil
}

// Отметить уведомление как прочитанное
func markNotificationAsRead(notificationID int) error {
	query := `UPDATE notifications SET is_read = 1 WHERE id = ?`
	_, err := db.Exec(query, notificationID)
	if err != nil {
		return fmt.Errorf("ошибка отметки уведомления как прочитанного: %v", err)
	}
	return nil
}

// Очистить все уведомления учителя
func clearTeacherNotifications(teacherID int64) error {
	query := `DELETE FROM notifications WHERE teacher_id = ?`
	_, err := db.Exec(query, teacherID)
	if err != nil {
		return fmt.Errorf("ошибка очистки уведомлений: %v", err)
	}
	return nil
}
