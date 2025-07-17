.PHONY: build install server client install-server install-client uninstall start stop restart status update clean gen-cert

# Основные команды сборки
build: server client

server:
	go build -o bin/dispatcher-server ./cmd/dispatcher-server

client:
	go build -o bin/dispatcher-client ./cmd/dispatcher-client

# Генерация сертификатов
gen-cert:
	@echo "Generating SSL certificates for QUIC..."
	mkdir -p config
	openssl req -x509 -newkey rsa:2048 -nodes -keyout config/localhost-key.pem -out config/localhost.pem -subj "/CN=localhost" -days 365
	@echo "SSL certificates generated successfully"

# Установка бинарников и сервисов
install: gen-cert install-server install-client
	@echo "Creating configuration files..."
	sudo touch /etc/dispatcher/server.yaml
	sudo touch /etc/dispatcher/client.yaml

install-server: server
	@echo "Installing server..."
	sudo cp bin/dispatcher-server /usr/local/bin/
	sudo mkdir -p /etc/dispatcher
	sudo mkdir -p /etc/dispatcher/config
	sudo cp config/localhost.pem /etc/dispatcher/config/
	sudo cp config/localhost-key.pem /etc/dispatcher/config/
	sudo cp deploy/dispatcher-server.service /etc/systemd/system/
	sudo systemctl daemon-reload
	@echo "Server installed. Use 'make start-server' to start the service."

install-client: client
	@echo "Installing client..."
	sudo cp bin/dispatcher-client /usr/local/bin/
	sudo mkdir -p /etc/dispatcher
	sudo cp deploy/dispatcher-client.service /etc/systemd/system/
	sudo systemctl daemon-reload
	@echo "Client installed. Use 'make start-client' to start the service."

uninstall:
	sudo systemctl stop dispatcher-server.service || true
	sudo systemctl stop dispatcher-client.service || true
	sudo systemctl disable dispatcher-server.service || true
	sudo systemctl disable dispatcher-client.service || true
	sudo rm -f /etc/systemd/system/dispatcher-server.service
	sudo rm -f /etc/systemd/system/dispatcher-client.service
	sudo rm -f /usr/local/bin/dispatcher-server
	sudo rm -f /usr/local/bin/dispatcher-client
	sudo systemctl daemon-reload
	@echo "Dispatcher uninstalled"

# Управление сервисами
start-server:
	sudo systemctl start dispatcher-server.service
	@echo "Server started"

start-client:
	sudo systemctl start dispatcher-client.service
	@echo "Client started"

start: start-server start-client

stop-server:
	sudo systemctl stop dispatcher-server.service
	@echo "Server stopped"

stop-client:
	sudo systemctl stop dispatcher-client.service
	@echo "Client stopped"

stop: stop-server stop-client

restart-server: stop-server start-server

restart-client: stop-client start-client

restart: restart-server restart-client

enable-server:
	sudo systemctl enable dispatcher-server.service
	@echo "Server enabled to start at boot"

enable-client:
	sudo systemctl enable dispatcher-client.service
	@echo "Client enabled to start at boot"

enable: enable-server enable-client

status:
	@echo "Server status:"
	@sudo systemctl status dispatcher-server.service || true
	@echo "Client status:"
	@sudo systemctl status dispatcher-client.service || true

# Обновление конфигурации и перезапуск
update-config-server:
	@echo "Updating server configuration and restarting service..."
	sudo systemctl restart dispatcher-server.service
	@echo "Server configuration updated and service restarted"

update-config-client:
	@echo "Updating client configuration and restarting service..."
	sudo systemctl restart dispatcher-client.service
	@echo "Client configuration updated and service restarted"

update-config: update-config-server update-config-client

# Очистка
clean:
	rm -rf bin/
