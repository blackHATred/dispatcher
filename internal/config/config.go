package config

import (
	"flag"
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
)

// ServerConfig содержит конфигурацию сервера
type ServerConfig struct {
	Network struct {
		ListenIP   string `yaml:"listenIP"`
		ListenPort int    `yaml:"listenPort"`
		SseIP      string `yaml:"sseIP"`
		SsePort    int    `yaml:"ssePort"`
		Cors       string `yaml:"cors"`
	} `yaml:"network"`

	SSL struct {
		CertFile string `yaml:"certFile"`
		KeyFile  string `yaml:"keyFile"`
	} `yaml:"ssl"`

	Processing struct {
		FilterRadius float64 `yaml:"filterRadius"`
	} `yaml:"processing"`
}

// ClientConfig содержит конфигурацию клиента
type ClientConfig struct {
	Network struct {
		ServerIP   string `yaml:"serverIP"`
		ServerPort int    `yaml:"serverPort"`
		ListenIP   string `yaml:"listenIP"`
		ListenPort int    `yaml:"listenPort"`
	} `yaml:"network"`

	Processing struct {
		FilterRadius float64 `yaml:"filterRadius"`
		VoxelSize    float64 `yaml:"voxelSize"`
	} `yaml:"processing"`
}

// LoadServerConfig загружает конфигурацию сервера из файла и флагов
func LoadServerConfig() (*ServerConfig, error) {
	configPath := flag.String("config", "/etc/dispatcher/server.yaml", "Путь к файлу конфигурации")

	// Сетевые настройки
	listenIP := flag.String("ip", "", "IP для прослушивания QUIC")
	listenPort := flag.Int("port", 0, "Порт для прослушивания QUIC")
	ssePort := flag.Int("sse-port", 0, "Порт SSE")
	sseIP := flag.String("sse-ip", "", "IP SSE")
	cors := flag.String("cors", "", "Значение Access-Control-Allow-Origin для CORS")

	// SSL настройки
	certFile := flag.String("cert", "", "Путь к файлу сертификата")
	keyFile := flag.String("key", "", "Путь к файлу ключа")

	// Настройки обработки
	filterRadius := flag.Float64("filter-radius", -1, "Радиус фильтрации точек у центра (0 - отключить фильтр)")

	flag.Parse()

	// Проверяем существование директории
	configDir := filepath.Dir(*configPath)
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		// Создаем директорию, если она не существует
		if err := os.MkdirAll(configDir, 0755); err != nil {
			return nil, fmt.Errorf("невозможно создать директорию для конфигурации: %w", err)
		}
	}

	// Создаем дефолтную конфигурацию
	config := &ServerConfig{}
	config.Network.ListenIP = "0.0.0.0"
	config.Network.ListenPort = 8081
	config.Network.SseIP = "0.0.0.0"
	config.Network.SsePort = 8080
	config.Network.Cors = "*"

	// Используем пути к сертификатам внутри /etc/dispatcher/config
	config.SSL.CertFile = "/etc/dispatcher/config/localhost.pem"
	config.SSL.KeyFile = "/etc/dispatcher/config/localhost-key.pem"

	config.Processing.FilterRadius = 0.05

	// Пытаемся загрузить конфигурацию из файла
	if _, err := os.Stat(*configPath); err == nil {
		data, err := os.ReadFile(*configPath)
		if err != nil {
			return nil, fmt.Errorf("невозможно прочитать файл конфигурации: %w", err)
		}

		if err := yaml.Unmarshal(data, config); err != nil {
			return nil, fmt.Errorf("невозможно распарсить YAML конфигурацию: %w", err)
		}
	} else {
		// Если файл не существует, создаем его
		data, err := yaml.Marshal(config)
		if err != nil {
			return nil, fmt.Errorf("невозможно сериализовать конфигурацию: %w", err)
		}

		if err := os.WriteFile(*configPath, data, 0644); err != nil {
			return nil, fmt.Errorf("невозможно записать конфигурацию в файл: %w", err)
		}

		fmt.Printf("Создан файл конфигурации: %s\n", *configPath)
	}

	// Параметры командной строки имеют приоритет над конфигурационным файлом
	if *listenIP != "" {
		config.Network.ListenIP = *listenIP
	}
	if *listenPort != 0 {
		config.Network.ListenPort = *listenPort
	}
	if *sseIP != "" {
		config.Network.SseIP = *sseIP
	}
	if *ssePort != 0 {
		config.Network.SsePort = *ssePort
	}
	if *cors != "" {
		config.Network.Cors = *cors
	}
	if *certFile != "" {
		config.SSL.CertFile = *certFile
	}
	if *keyFile != "" {
		config.SSL.KeyFile = *keyFile
	}
	if *filterRadius != -1 {
		config.Processing.FilterRadius = *filterRadius
	}

	return config, nil
}

// LoadClientConfig загружает конфигурацию клиента из файла и флагов
func LoadClientConfig() (*ClientConfig, error) {
	configPath := flag.String("config", "/etc/dispatcher/client.yaml", "Путь к файлу конфигурации")

	// Сетевые настройки
	serverIP := flag.String("server-ip", "", "IP удалённого сервера для QUIC соединения")
	serverPort := flag.Int("server-port", 0, "Порт удалённого сервера для QUIC соединения")
	listenPort := flag.Int("port", 0, "Порт для UDP сервера")
	listenIP := flag.String("ip", "", "IP для прослушивания UDP")

	// Настройки обработки
	filterRadius := flag.Float64("filter-radius", -1, "Радиус фильтрации точек у центра (0 - отключить фильтр)")
	voxelSize := flag.Float64("voxel-size", -1, "Размер вокселя для компрессора")

	flag.Parse()

	// Проверяем существование директории
	configDir := filepath.Dir(*configPath)
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		// Создаем директорию, если она не существует
		if err := os.MkdirAll(configDir, 0755); err != nil {
			return nil, fmt.Errorf("невозможно создать директорию для конфигурации: %w", err)
		}
	}

	// Создаем дефолтную конфигурацию
	config := &ClientConfig{}
	config.Network.ServerIP = "localhost"
	config.Network.ServerPort = 8081
	config.Network.ListenIP = "0.0.0.0"
	config.Network.ListenPort = 2368
	config.Processing.FilterRadius = 0.5
	config.Processing.VoxelSize = 0.05

	// Пытаемся загрузить конфигурацию из файла
	if _, err := os.Stat(*configPath); err == nil {
		data, err := os.ReadFile(*configPath)
		if err != nil {
			return nil, fmt.Errorf("невозможно прочитать файл конфигурации: %w", err)
		}

		if err := yaml.Unmarshal(data, config); err != nil {
			return nil, fmt.Errorf("невозможно распарсить YAML конфигурацию: %w", err)
		}
	} else {
		// Если файл не существует, создаем его
		data, err := yaml.Marshal(config)
		if err != nil {
			return nil, fmt.Errorf("невозможно сериализовать конфигурацию: %w", err)
		}

		if err := os.WriteFile(*configPath, data, 0644); err != nil {
			return nil, fmt.Errorf("невозможно записать конфигурацию в файл: %w", err)
		}

		fmt.Printf("Создан файл конфигурации: %s\n", *configPath)
	}

	// Параметры командной строки имеют приоритет над конфигурационным файлом
	if *serverIP != "" {
		config.Network.ServerIP = *serverIP
	}
	if *serverPort != 0 {
		config.Network.ServerPort = *serverPort
	}
	if *listenIP != "" {
		config.Network.ListenIP = *listenIP
	}
	if *listenPort != 0 {
		config.Network.ListenPort = *listenPort
	}
	if *filterRadius != -1 {
		config.Processing.FilterRadius = *filterRadius
	}
	if *voxelSize != -1 {
		config.Processing.VoxelSize = *voxelSize
	}

	return config, nil
}
