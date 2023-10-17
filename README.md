# archiveApi

Сервер работы с архивом
    1. отображение информации об архиве и его содержимом (http://localhost:8080/api/archive/information)
    2. архивирование файлов в zip (http://localhost:8080/api/archive/files)
    3. рассылка файлов на почту (http://localhost:8080/api/mail/file)

Для запуска 1 и 2 функции можно использовать команду `go run main.go`

Для запуска всех 3х функций используйте команду:

`EMAIL_USERNAME=your_email@gmail.com EMAIL_PASSWORD=your_password SMTP_HOST=smtp.gmail.com SMTP_PORT=587 go run main.go`

замените поля - `your_email@gmail.com` и `your_password` на свои данные если вы используете gmail.com
Если вы используете другой почтовый червер, так же замените `smtp.gmail.com` и `587` на настройки своего почтового сервера


