# 🗓️ Tutor Scheduler Bot

Умный Telegram-бот для управления расписанием занятий с автоматическими уведомлениями и удобным интерфейсом.

![Bot Preview](screenshots/menu_example.png)

## 🌟 Особенности

### 👨🏫 Для преподавателей
- 📅 Интерактивный календарь для управления расписанием
- 👥 Просмотр списка учеников
- 🔔 Уведомления о новых записях и отменах
- ⏰ Автоматические напоминания о занятиях
- 📊 Статистика занятости

### 👨🎓 Для учеников
- 🔍 Поиск доступных слотов
- 📝 Запись на занятия в один клик
- 🗑️ Отмена записей
- 🔔 Напоминания о предстоящих занятиях
- 📱 Адаптивное меню

## 🛠 Технологии

- **Язык программирования**: Go 1.19+
- **База данных**: SQLite3
- **Основные библиотеки**:
  - `go-telegram-bot-api` - работа с Telegram API
  - `mattn/go-sqlite3` - драйвер для SQLite
- **Архитектура**: Модульная структура с разделением на:
  - `main.go` - инициализация и запуск
  - `database.go` - работа с БД
  - `handlers.go` - обработчики команд
  - `notifications.go` - система уведомлений
  - `models.go` - модели данных

## 🚀 Быстрый старт

### Требования
- Go 1.19+
- SQLite3
- Telegram-бот токен

### Установка
1. Клонируйте репозиторий:
   ```bash
   git clone https://github.com/yourusername/tutor-scheduler-bot.git

2. Установите зависимости:
   ```bash
   go mod download

3. Настройте окружение:
   ```bash
   export TELEGRAM_BOT_TOKEN="ваш_токен"
   const teacherID = ВАШ_ID // Ваш Telegram ID

4. Запустите бота:
   ```bash
   go run main.go

## Скриншоты интерфейса 📸
| Скриншот              | Описание            |
|-----------------------|---------------------|
| <img src="screenshots/teacher_menu.png" alt="Teacher Menu" width="300"> | Меню преподавателя |
| <img src="screenshots/calendar.png" alt="Calendar" width="300">         | Календарь слотов   |
| <img src="screenshots/student_booking.png" alt="Student Booking" width="300"> | Запись ученика |
| <img src="screenshots/notifications.png" alt="Notifications" width="300"> | Уведомления     |

## Основные команды
| Команда       | Описание               |
|---------------|-----------------------|
| `/start`      | Начало работы         |
| `/schedule`   | Управление расписанием|
| `/students`   | Список учеников       |
| `/book`       | Записаться на занятие |
| `/mybookings` | Мои записи            |
| `/cancel`     | Отмена записи         |
