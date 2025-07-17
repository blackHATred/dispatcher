## Установка и настройка
Подключите orange pi 5 max (или любое другое устройство, отвечающее за принятие данных с lidar) к интернету.
Склонируйте репозиторий следующей командой:

### 1. Клонирование репозитория
```bash
cd ~ && git clone https://github.com/blackHATred/dispatcher
cd dispatcher
```

### 2. Настройка сетевого буфера

Так как получение данных от lidar происходит по UDP, а передача облака точек по QUIC, то настоятельно
рекомендуется повысить сетевой буфер в ОС. Для этого выполните следующие команды:
```bash
sudo sysctl -w net.core.rmem_max=30146560
sudo sysctl -w net.core.wmem_max=30146560
sudo sysctl -w net.core.rmem_default=30146560
sudo sysctl -w net.core.wmem_default=30146560
```
Рекомендуется иметь 25 МБ буфера. Так как для систем BSD / Darwin рекомендуется закладывать +15%, то
получаем 26214400*1.15 = 30146560 Байт.

**Важно**: Эти настройки не сохраняются после перезагрузки системы. Для постоянного сохранения необходимо добавить их в файл конфигурации:

Для большинства Linux систем:
```bash
echo "net.core.rmem_max=30146560" | sudo tee -a /etc/sysctl.conf
echo "net.core.wmem_max=30146560" | sudo tee -a /etc/sysctl.conf
echo "net.core.rmem_default=30146560" | sudo tee -a /etc/sysctl.conf
echo "net.core.wmem_default=30146560" | sudo tee -a /etc/sysctl.conf
sudo sysctl -p
```

Для более новых дистрибутивов Linux можно использовать:
```bash
echo "net.core.rmem_max=30146560" | sudo tee -a /etc/sysctl.d/99-custom.conf
echo "net.core.wmem_max=30146560" | sudo tee -a /etc/sysctl.d/99-custom.conf
echo "net.core.rmem_default=30146560" | sudo tee -a /etc/sysctl.d/99-custom.conf
echo "net.core.wmem_default=30146560" | sudo tee -a /etc/sysctl.d/99-custom.conf
sudo sysctl -p /etc/sysctl.d/99-custom.conf
```

После добавления настроек в файл конфигурации, они будут применяться при каждой загрузке системы.

### 3. Генерация SSL сертификатов

QUIC протокол требует использования SSL сертификатов. Для генерации самоподписанных сертификатов выполните:

```bash
make gen-cert
```

Эта команда создаст самоподписанный сертификат и ключ в директории `config/`.

### 4. Компиляция и установка

Для сборки и установки приложений выполните:

```bash
make build      # только сборка
make install    # сборка, генерация сертификатов и установка как сервиса
```

**Примечание**: Команда `make install` автоматически включает генерацию SSL сертификатов, поэтому отдельно выполнять `make gen-cert` не нужно.

### 5. Конфигурация

После установки автоматически создаются конфигурационные файлы:

- Для сервера: `/etc/dispatcher/server.yaml`
- Для клиента: `/etc/dispatcher/client.yaml`

Вы можете редактировать эти файлы для настройки приложений:

#### Пример конфигурации сервера (server.yaml):
```yaml
network:
  listenIP: 0.0.0.0         # IP для прослушивания QUIC
  listenPort: 8081          # Порт для прослушивания QUIC
  sseIP: 0.0.0.0            # IP для SSE
  ssePort: 8080             # Порт для SSE
  cors: "*"                 # CORS настройка
ssl:
  certFile: /etc/dispatcher/config/localhost.pem   # Путь к сертификату
  keyFile: /etc/dispatcher/config/localhost-key.pem # Путь к ключу
processing:
  filterRadius: 0.05        # Радиус фильтрации точек
```

#### Пример конфигурации клиента (client.yaml):
```yaml
network:
  serverIP: 192.168.1.100   # IP сервера
  serverPort: 8081          # Порт сервера
  listenIP: 0.0.0.0         # IP для прослушивания UDP
  listenPort: 2368          # Порт для прослушивания UDP (стандартный порт LiDAR)
processing:
  filterRadius: 0.5         # Радиус фильтрации точек
  voxelSize: 0.05           # Размер вокселя для компрессора
```

### 6. Управление сервисами

После установки вы можете управлять сервисами с помощью следующих команд (выполнять из директории проекта):

```bash
make start           # Запустить и сервер, и клиент
make start-server    # Запустить только сервер
make start-client    # Запустить только клиент

make stop            # Остановить и сервер, и клиент
make stop-server     # Остановить только сервер
make stop-client     # Остановить только клиент

make restart         # Перезапустить и сервер, и клиент
make status          # Проверить статус сервисов

# Добавить в автозагрузку
make enable          # Добавить в автозагрузку и сервер, и клиент
make enable-server   # Добавить в автозагрузку только сервер
make enable-client   # Добавить в автозагрузку только клиент

# После изменения конфигурации
make update-config           # Перезапустить оба сервиса для применения изменений
make update-config-server    # Перезапустить только сервер
make update-config-client    # Перезапустить только клиент

# Удалить сервисы
make uninstall       # Удалить все сервисы и бинарные файлы
```

### 7. Параметры командной строки

Вы также можете переопределить параметры из конфигурационного файла с помощью аргументов командной строки:

#### Для сервера:
```bash
dispatcher-server --config=/path/to/config.yaml --ip=0.0.0.0 --port=8081 --sse-ip=0.0.0.0 --sse-port=8080 --cors="*" --cert=/path/to/cert.pem --key=/path/to/key.pem --filter-radius=0.05
```

#### Для клиента:
```bash
dispatcher-client --config=/path/to/config.yaml --server-ip=192.168.1.100 --server-port=8081 --ip=0.0.0.0 --port=2368 --filter-radius=0.5 --voxel-size=0.05
```

## Быстрый старт

Для быстрого старта с минимальными усилиями:

```bash
# Клонировать репозиторий
git clone https://github.com/blackHATred/dispatcher
cd dispatcher

# Установить сервисы
make install

# Настроить сетевой буфер (постоянно)
echo "net.core.rmem_max=30146560" | sudo tee -a /etc/sysctl.conf
echo "net.core.wmem_max=30146560" | sudo tee -a /etc/sysctl.conf
echo "net.core.rmem_default=30146560" | sudo tee -a /etc/sysctl.conf
echo "net.core.wmem_default=30146560" | sudo tee -a /etc/sysctl.conf
sudo sysctl -p

# Отредактировать конфигурационные файлы (при необходимости)
sudo nano /etc/dispatcher/server.yaml
sudo nano /etc/dispatcher/client.yaml

# Добавить сервисы в автозагрузку
make enable

# Запустить сервисы
make start

# Проверить статус
make status
```
