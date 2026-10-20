package main

import (
	_ "embed"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"unix-hardware-agent/services" // Убедись, что имя модуля совпадает с твоим go.mod
)

// Упаковываем бинарник smartctl для Linux/Unix
// Перед сборкой под конкретную ОС (например, Linux x86_64), в корень проекта нужно положить соответствующий файл 'smartctl'
//go:embed smartctl
var smartctlBytes []byte

func initLogger() {
	logsEnable := os.Getenv("HOSTPULSE_LOGS_ENABLE")

	// Если передано "false", пишем только в консоль (тихий режим)
	if logsEnable == "false" {
		log.SetOutput(os.Stdout)
		log.SetFlags(log.LstdFlags | log.Lshortfile)
		log.SetPrefix("")
		fmt.Println(" [ℹ️ SYSTEM] Агент запущен в тихом режиме. Запись файла логов отключена.")
		return
	}

	// Unix-way: Храним логи в стандартном системном месте /var/log/
	// Если папка недоступна (например, запуск без root при локальной отладке), пишем рядом с бинарником
	logDir := "/var/log/hostpulse"
	if err := os.MkdirAll(logDir, 0755); err != nil {
		exePath, err := os.Executable()
		if err != nil {
			exePath = "."
		}
		logDir = filepath.Dir(exePath)
	}

	logFilePath := filepath.Join(logDir, "agent_hardware.log")

	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		fmt.Printf(" [❌ ERROR] Не удалось создать файл лога: %v\n", err)
		log.SetOutput(os.Stdout)
		return
	}

	// Дублируем вывод в консоль и в файл лога
	mw := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(mw)

	// Перенаправляем системные потоки вывода в файл лога
	os.Stdout = logFile
	os.Stderr = logFile
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.SetPrefix("")
}

func main() {
	// Инициализация логгера
	initLogger()

	// Передаем вкомпилированный бинарник smartctl в сервисы
	services.SetSmartctlBinary(smartctlBytes)

	fmt.Println("\n=======================================================")
	fmt.Println("=== Агент мониторинга железа HostPulse под Unix запущен ===")
	fmt.Println("=======================================================")

	metricsURL := os.Getenv("HOSTPULSE_HARDWARE_URL")
	if metricsURL == "" {
		metricsURL = "https://zedform.kz"
	}

	agentToken := os.Getenv("HOSTPULSE_TOKEN")
	if agentToken == "" {
		fmt.Println(" [⚠️ WARNING] HOSTPULSE_TOKEN не задан. Используется дефолтный токен.")
		agentToken = "default_hardware_token_123"
	}

	fmt.Printf(" [INFO] Базовый эндпоинт: %s\n", metricsURL)

	// Разовый запуск паспорта инвентаризации системы (CPU, RAM, Motherboard, OS)
	services.SendStaticHardwareInfo(metricsURL, agentToken)

	// Запускаем воркеры параллельно в независимых горутинах
	go services.StartCpuPoller(metricsURL, agentToken)
	go services.StartDiskPoller(metricsURL, agentToken)
	go services.StartMemoryHardwarePoller(metricsURL, agentToken)
	go services.StartPowerSupplyHardwarePoller(metricsURL, agentToken)
	go services.StartNetworkHardwarePoller(metricsURL, agentToken)

	// Блокируем поток для бесконечной фоновой работы службы
	select {}
}
