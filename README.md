# Сервис для сокращения URL

Pet проект для яндекс практикум

## Список команд
- POST / - принимает в теле запроса строку URL для сокращения и возвращает ответ с кодом 201 и сокращённым URL в виде текстовой строки в теле
- GET /{id} - принимает в качестве URL-параметра идентификатор сокращённого URL и возвращает ответ с кодом 307 и оригинальным URL в HTTP-заголовке Location
- POST /api/shorten - принимает в теле запроса JSON-объект {"url":"<some_url>"} и возвращает в ответ объект {"result":"<shorten_url>"}
- POST /api/shorten/batch - принимает в теле запроса множество URL для сокращения в формате JSON-объектов и возвращает в ответ множество JSON-объектов
- DELETE /api/user/urls - принимает список идентификаторов сокращённых URL для удаления в формате: [ "a", "b", ...] и возвращает HTTP-статус 202 Accepted
- GET /api/user/urls - возвращает все URL, сокращенные пользователем в формате множества JSON-объектов

# Запуск

Собрать исполняемый файл в директории /cmd/shortener, запустить сервер.
В качестве клиента можно использовать Postman.

# База данных

Используется PostqreSQL. Строка соединения по умолчанию:
"user=habruser password=habr host=localhost port=5432 dbname=habrdb sslmode=disable". 

# Конфигурация приложения

Способы получения значений конфигурации в порядке возрастания приоритета:
 - из файла cmd/shortener/config.json
 - флаги командной строки
 - переменные окружения

