# Go survey bot

## Запуск
- Склонировать репо
- положить два файла
  - ```credentials.json``` - креды сервисного аккаунта гугл
  - ```.env``` - пример ниже
- ```docker-compose up```
- Done!

### .env
```
export GOOGLE_SHEET_NAME="{название листа excel}"
export GOOGLE_SPREADSHEET_ID="{id документа}"
export TELEGRAM_BOT_TOKEN="{токен telegram bot api}"
```
