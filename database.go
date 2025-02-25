package main

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// Инициализация базы данных
func InitDB() (*sql.DB, error) {
	db, err := sql.Open("sqlite3", "./schedule.db")
	if err != nil {
		return nil, fmt.Errorf("ошибка открытия базы данных: %v", err)
	}

	// Проверка соединения
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ошибка подключения к базе данных: %v", err)
	}

	// Создание таблиц
	if err := createTables(db); err != nil {
		return nil, fmt.Errorf("ошибка создания таблиц: %v", err)
	}

	return db, nil
}

// Создание таблиц в базе данных
func createTables(db *sql.DB) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS users (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            telegram_id INTEGER UNIQUE,
            role TEXT CHECK(role IN ('teacher', 'student')),
            username TEXT,
            contact TEXT
        )`,
		`CREATE TABLE IF NOT EXISTS schedules (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            teacher_id INTEGER,
            start_time TEXT,
            end_time TEXT,
            notified BOOLEAN DEFAULT 0,
            status TEXT CHECK(status IN ('free', 'booked')),
            student_id INTEGER,
            direction TEXT,
            FOREIGN KEY(teacher_id) REFERENCES users(telegram_id)
        )`,
		// Новая таблица для уведомлений
		`CREATE TABLE IF NOT EXISTS notifications (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            teacher_id INTEGER,
            message TEXT,
            created_at TEXT,
            is_read BOOLEAN DEFAULT 0,
            FOREIGN KEY(teacher_id) REFERENCES users(telegram_id)
        )`,
	}

	for _, query := range queries {
		if _, err := db.Exec(query); err != nil {
			return fmt.Errorf("ошибка выполнения запроса %q: %v", query, err)
		}
	}
	return nil
}

// Получение списка учеников для учителя
func getTeacherStudents(teacherID int64) ([]BookingNotification, error) {
	query := `SELECT s.student_id, u.username, s.start_time, s.direction 
		FROM schedules s
		JOIN users u ON s.student_id = u.telegram_id
		WHERE s.teacher_id = ? AND s.status = 'booked'`
	rows, err := db.Query(query, teacherID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var students []BookingNotification
	for rows.Next() {
		var s BookingNotification
		rows.Scan(&s.StudentID, &s.StudentUsername, &s.StartTime, &s.Direction)
		students = append(students, s)
	}
	return students, nil
}

// Регистрация нового пользователя
func registerUser(telegramID int64, role, username string) error {
	query := `INSERT INTO users (telegram_id, role, username) VALUES (?, ?, ?)`
	_, err := db.Exec(query, telegramID, role, username)
	if err != nil {
		return fmt.Errorf("ошибка регистрации пользователя: %v", err)
	}
	return nil
}

// Проверка существования пользователя
func userExists(telegramID int64) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM users WHERE telegram_id = ?)`
	err := db.QueryRow(query, telegramID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("ошибка проверки пользователя: %v", err)
	}
	return exists, nil
}

// Получение пользователя по Telegram ID
func getUser(telegramID int64) (*User, error) {
	var user User
	query := `SELECT id, telegram_id, role, username, contact 
		FROM users WHERE telegram_id = ?`
	err := db.QueryRow(query, telegramID).Scan(
		&user.ID,
		&user.TelegramID,
		&user.Role,
		&user.Username,
		&user.Contact)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения пользователя: %v", err)
	}
	return &user, nil
}

// Добавление нового слота в расписание
func addScheduleSlot(teacherID int64, startTime, endTime string) error {
	// Проверка существования слота
	var exists bool
	err := db.QueryRow(
		`SELECT EXISTS(
            SELECT 1 FROM schedules 
            WHERE teacher_id = ? 
            AND start_time = ?
        )`, teacherID, startTime).Scan(&exists)

	if err != nil {
		return fmt.Errorf("ошибка проверки слота: %v", err)
	}

	if exists {
		return fmt.Errorf("слот уже существует")
	}

	// Добавление нового слота
	_, err = db.Exec(
		`INSERT INTO schedules 
        (teacher_id, start_time, end_time, status) 
        VALUES (?, ?, ?, 'free')`,
		teacherID, startTime, endTime)

	if err != nil {
		return fmt.Errorf("ошибка добавления слота: %v", err)
	}
	return nil
}

func getSlotsForDate(date time.Time) ([]Schedule, error) {
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC).Format(time.RFC3339)
	endOfDay := time.Date(date.Year(), date.Month(), date.Day(), 23, 59, 59, 0, time.UTC).Format(time.RFC3339)

	query := `SELECT id, start_time, end_time, status 
        FROM schedules 
        WHERE teacher_id = ? 
        AND start_time BETWEEN ? AND ? 
        ORDER BY start_time`
	rows, err := db.Query(query, teacherID, startOfDay, endOfDay)
	if err != nil {
		return nil, fmt.Errorf("ошибка запроса слотов: %v", err)
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

func setUserLastMessage(chatID int64, msgID int) {
	_, err := db.Exec(
		`UPDATE users 
        SET last_message_id = ? 
        WHERE telegram_id = ?`,
		msgID, chatID)

	if err != nil {
	}
}

func getUserLastMessage(chatID int64) (int, bool) {
	var msgID int
	err := db.QueryRow(
		`SELECT last_message_id 
        FROM users 
        WHERE telegram_id = ?`,
		chatID).Scan(&msgID)

	if err != nil {
		if err != sql.ErrNoRows {
		}
		return 0, false
	}
	return msgID, true
}

// Получение расписания учителя
func getTeacherSchedule(teacherID int64) ([]Schedule, error) {

	query := `SELECT id, start_time, end_time, status 
        FROM schedules 
        WHERE teacher_id = ? 
        ORDER BY start_time`

	rows, err := db.Query(query, teacherID)
	if err != nil {
		return nil, fmt.Errorf("database error: %v", err)
	}
	defer rows.Close()

	var schedules []Schedule
	for rows.Next() {
		var s Schedule
		if err := rows.Scan(&s.ID, &s.StartTime, &s.EndTime, &s.Status); err != nil {
			continue
		}
		schedules = append(schedules, s)
	}

	return schedules, nil
}

// Получение записей ученика
func getStudentBookings(studentID int64) ([]Schedule, error) {
	rows, err := db.Query(`SELECT id, start_time, end_time, direction 
                           FROM schedules 
                           WHERE student_id = ? AND status = 'booked' 
                           ORDER BY start_time`, studentID)
	if err != nil {
		return nil, fmt.Errorf("ошибка запроса записей: %v", err)
	}
	defer rows.Close()

	var bookings []Schedule
	for rows.Next() {
		var s Schedule
		if err := rows.Scan(&s.ID, &s.StartTime, &s.EndTime, &s.Direction); err != nil {
			return nil, fmt.Errorf("ошибка сканирования записей: %v", err)
		}
		bookings = append(bookings, s)
	}
	return bookings, nil
}

// Обновление статуса слота (запись или отмена)
func updateScheduleStatus(scheduleID int64, status string, studentID int64, direction string) error {
	var studentIDValue interface{}
	if studentID == 0 {
		studentIDValue = nil // Устанавливаем NULL для свободного слота
	} else {
		studentIDValue = studentID
	}

	var directionValue interface{}
	if direction == "" {
		directionValue = nil // Устанавливаем NULL, если направление не указано
	} else {
		directionValue = direction
	}

	query := `UPDATE schedules 
              SET status = ?, student_id = ?, direction = ?
              WHERE id = ?`
	_, err := db.Exec(query, status, studentIDValue, directionValue, scheduleID)
	if err != nil {
		return fmt.Errorf("ошибка обновления статуса: %v", err)
	}
	return nil
}

// Удаление слота из расписания
func DeleteScheduleSlot(scheduleID int64) error {
	query := `DELETE FROM schedules WHERE id = ?`
	_, err := db.Exec(query, scheduleID)
	if err != nil {
		return fmt.Errorf("ошибка удаления слота: %v", err)
	}
	return nil
}

// Получение свободных слотов для записи
func getAvailableSlots() ([]Schedule, error) {
	query := `SELECT id, start_time 
		FROM schedules 
		WHERE status = 'free' AND start_time > ? 
		ORDER BY start_time`
	rows, err := db.Query(query, time.Now().Format(time.RFC3339))
	if err != nil {
		return nil, fmt.Errorf("ошибка запроса свободных слотов: %v", err)
	}
	defer rows.Close()

	var slots []Schedule
	for rows.Next() {
		var s Schedule
		if err := rows.Scan(&s.ID, &s.StartTime); err != nil {
			return nil, fmt.Errorf("ошибка сканирования слотов: %v", err)
		}
		slots = append(slots, s)
	}
	return slots, nil
}

// Получение информации о слоте
func getScheduleByID(scheduleID int64) (*Schedule, error) {
	var s Schedule
	query := `SELECT id, teacher_id, start_time, end_time, status, student_id, direction 
        FROM schedules WHERE id = ?`
	err := db.QueryRow(query, scheduleID).Scan(
		&s.ID, &s.TeacherID, &s.StartTime, &s.EndTime, &s.Status, &s.StudentID, &s.Direction)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("слот не найден")
	}
	if err != nil {
		return nil, fmt.Errorf("ошибка получения слота: %v", err)
	}
	return &s, nil
}
