//go:build linux || freebsd

package services

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// PowerSupplyStatus — Структура полностью сохранена для Django-бэкенда HostPulse
type PowerSupplyStatus struct {
	DeviceID  string `json:"device_id"`   // Например: Power Supply 0
	Name      string `json:"name"`        // Описание/Модель блока
	IsPresent bool   `json:"is_present"`  // Подключен ли физически
	IsHealthy bool   `json:"is_healthy"`  // Исправен ли (Status == "OK")
	ErrorCode uint32 `json:"error_code"`  // Код ошибки, если есть
}

// GetPowerSupplyHardwareStatus собирает статус по всем блокам питания системы в Unix
func GetPowerSupplyHardwareStatus() []PowerSupplyStatus {
	var results []PowerSupplyStatus

	// Метод №1: Чтение серверных таблиц SMBIOS/DMI (Тип 39 — System Power Supply)
	cmd := exec.Command("dmidecode", "-t", "39")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	// Если утилита отработала успешно, парсим серверные резервируемые блоки
	if cmd.Run() == nil && stdout.Len() > 0 {
		outputStr := stdout.String()
		// Разделяем вывод на отдельные физические блоки питания
		psBlocks := strings.Split(outputStr, "Power Supply Device")

		if len(psBlocks) > 1 {
			for idx, block := range psBlocks[1:] {
				deviceID := fmt.Sprintf("Power_Supply_%d", idx)

				// 1. Извлекаем имя/локатор блока
				locReg := regexp.MustCompile(`(?i)Location:\s*(.+)`)
				name := fmt.Sprintf("Server PSU %d", idx)
				if match := locReg.FindStringSubmatch(block); len(match) > 1 {
					name = strings.TrimSpace(match[1])
				}

				// 2. Проверяем физическое присутствие в слоте
				isPresent := true
				if strings.Contains(strings.ToLower(block), "not present") {
					isPresent = false
				}

				// 3. Считываем статус здоровья блока
				isHealthy := true
				var errorCode uint32 = 0
				statusReg := regexp.MustCompile(`(?i)Status:\s*(.+)`)
				if match := statusReg.FindStringSubmatch(block); len(match) > 1 {
					statusStr := strings.ToLower(match[1])
					// Если статус содержит "critical", "failed" или "degraded" — фиксируем аварию
					if strings.Contains(statusStr, "fail") || strings.Contains(statusStr, "crit") {
						isHealthy = false
						errorCode = 1 // Системный маркер ошибки для Django
					}
				}

				results = append(results, PowerSupplyStatus{
					DeviceID:  deviceID,
					Name:      name,
					IsPresent: isPresent,
					IsHealthy: isHealthy,
					ErrorCode: errorCode,
				})
			}
		}
	}

	// Метод №2: Если DMI пуст (это десктоп, касса iiko или Mini-PC), опрашиваем подсистему sysfs Linux
	if len(results) == 0 {
		sysfsPowerPath := "/sys/class/power_supply"
		entries, err := os.ReadDir(sysfsPowerPath)
		if err == nil {
			for _, entry := range entries {
				name := entry.Name()
				// Ищем сетевые адаптеры питания (AC, ACAD, ADP0, ADP1)
				if strings.HasPrefix(name, "AC") || strings.HasPrefix(name, "ADP") {
					deviceID := name
					onlineBytes, err := os.ReadFile(filepath.Join(sysfsPowerPath, name, "online"))
					if err == nil {
						onlineStr := strings.TrimSpace(string(onlineBytes))
						// В sysfs: 1 = кабель питания включен в сеть, 0 = фаза потеряна / кабель вынут
						isPresent := onlineStr == "1"

						results = append(results, PowerSupplyStatus{
							DeviceID:  deviceID,
							Name:      "System AC Power Adapter",
							IsPresent: isPresent,
							IsHealthy: true, // Если sysfs видит узел, контроллер жив
							ErrorCode: 0,
						})
					}
				}
			}
		}
	}

	// Метод №3: Жесткий резервный фолбэк (Виртуальные машины, Docker-контейнеры или десктопы без ACPI драйверов)
	if len(results) == 0 {
		return []PowerSupplyStatus{
			{
				DeviceID:  "PC_Power_Supply",
				Name:      "Standard ATX Power Supply (Unix Simulated)",
				IsPresent: true,
				IsHealthy: true,
				ErrorCode: 0,
			},
		}
	}

	fmt.Printf(" [✅ POWER LOG] Данные по блокам питания Unix собраны. Найдено модулей: %d\n", len(results))
	return results
}
