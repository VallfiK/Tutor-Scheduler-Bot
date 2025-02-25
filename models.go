package main

import (
	"database/sql"
	"time"
)

// User представляет пользователя (учителя или ученика)
type User struct {
	ID         int            // Уникальный идентификатор пользователя
	TelegramID int64          // ID пользователя в Telegram
	Role       string         // Роль: "teacher" или "student"
	Username   sql.NullString // Имя пользователя в Telegram (может быть NULL)
	Contact    sql.NullString // Контактная информация (может быть NULL)
}

// Schedule представляет слот в расписании
type Schedule struct {
	ID        int            // Уникальный идентификатор слота
	TeacherID int64          // ID учителя
	StartTime string         // Время начала занятия (в формате RFC3339)
	EndTime   string         // Время окончания занятия (в формате RFC3339)
	Status    string         // Статус: "free" или "booked"
	StudentID sql.NullInt64  // ID ученика (может быть NULL, если слот свободен)
	Direction sql.NullString // Направление (может быть NULL)
}

// BookingNotification представляет уведомление о новой записи
type BookingNotification struct {
	ID              int    // Уникальный идентификатор записи
	TeacherID       int64  // ID учителя
	StudentID       int64  // ID ученика
	StartTime       string // Время начала занятия
	EndTime         string // Время окончания занятия
	Direction       string // Направление занятия
	StudentUsername string // Имя пользователя ученика
}

// CancellationNotification представляет уведомление об отмене записи
type CancellationNotification struct {
	ID              int    // Уникальный идентификатор отмены
	TeacherID       int64  // ID учителя
	StudentID       int64  // ID ученика
	StartTime       string // Время начала занятия
	EndTime         string // Время окончания занятия
	StudentUsername string // Имя пользователя ученика
}

// WeekSchedule представляет расписание на неделю
type WeekSchedule struct {
	WeekStart time.Time // Начало недели
	WeekEnd   time.Time // Конец недели
	Slots     []Slot    // Слоты расписания
}

// Slot представляет один слот в расписании
type Slot struct {
	ID        int       // Уникальный идентификатор слота
	StartTime time.Time // Время начала
	EndTime   time.Time // Время окончания
	Status    string    // Статус: "free" или "booked"
}

// TeacherStats представляет статистику учителя
type TeacherStats struct {
	TotalSlots    int // Общее количество слотов
	BookedSlots   int // Количество забронированных слотов
	FreeSlots     int // Количество свободных слотов
	Cancellations int // Количество отмен
}

// StudentStats представляет статистику ученика
type StudentStats struct {
	TotalBookings  int // Общее количество записей
	ActiveBookings int // Количество активных записей
	Cancellations  int // Количество отмен
}

// NotificationSettings представляет настройки уведомлений
type NotificationSettings struct {
	UserID              int64 // ID пользователя
	EnableReminders     bool  // Включены ли напоминания
	EnableNewBookings   bool  // Включены ли уведомления о новых записях
	EnableCancellations bool  // Включены ли уведомления об отменах
}

// ErrorLog представляет запись об ошибке
type ErrorLog struct {
	ID        int       // Уникальный идентификатор ошибки
	Timestamp time.Time // Время возникновения ошибки
	Message   string    // Сообщение об ошибке
	Details   string    // Детали ошибки
}

type Notification struct {
	ID        int    // Уникальный идентификатор уведомления
	TeacherID int64  // ID учителя
	Message   string // Текст уведомления
	CreatedAt string // Время создания (RFC3339)
	IsRead    bool   // Прочитано или нет
}
