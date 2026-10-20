//go:build linux || freebsd

package services

import (
	"bytes"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// MemorySlotStatus — Структура полностью сохранена под ваш Django бэкенд
type MemorySlotStatus struct {
	SlotLabel           string `json:"slot_label"`           // Например: DIMM_A1
	Manufacturer        string `json:"manufacturer"`         // Например: Samsung, SK Hynix
	CapacityGB          uint32 `json:"capacity_gb"`          // Объем планки в ГБ
	SpeedMhz            uint32 `json:"speed_mhz"`            // Частота работы
	CorrectableErrors   uint64 `json:"correctable_errors"`   // Исправленные ошибки ECC
	UncorrectableErrors uint64 `json:"uncorrectable_errors"` // Критические ошибки ECC
}

// GetMemoryHardwareStatus собирает детальные данные по планкам RAM и ECC ошибкам
func GetMemoryHardwareStatus() []MemorySlotStatus {
	var results []MemorySlotStatus

	// Вызываем стандартную утилиту чтения таблиц SMBIOS/DMI
	cmd := exec.Command("dmidecode", "-t", "memory")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	err := cmd.Run()
	if err != nil || stdout.Len() == 0 {
		fmt.Printf(" [❌ RAM ERROR] Не удалось выполнить dmidecode (требуются права root): %v\n", err)
		return getFallbackMemoryStatus()
	}

	// Сначала ищем глобальные счетчики ECC ошибок в начале вывода dmidecode (если поддерживается платформой)
	var globalCorrectable, globalUncorrectable uint64
	outputStr := stdout.String()

	// Поиск ошибок (может присутствовать в блоке "Memory Error Information")
	eccReg := regexp.MustCompile(`(?i)Correctable Error Count:\s*(\d+)`)
	if match := eccReg.FindStringSubmatch(outputStr); len(match) > 1 {
		globalCorrectable, _ = strconv.ParseUint(match[1], 10, 64)
	}
	ueccReg := regexp.MustCompile(`(?i)Uncorrectable Error Count:\s*(\d+)`)
	if match := ueccReg.FindStringSubmatch(outputStr); len(match) > 1 {
		globalUncorrectable, _ = strconv.ParseUint(match[1], 10, 64)
	}

	// Разделяем вывод на отдельные блоки "Memory Device" (каждый блок — один физический слот)
	devices := strings.Split(outputStr, "Memory Device")
	if len(devices) <= 1 {
		return getFallbackMemoryStatus()
	}

	for _, device := range devices[1:] {
		// Отсекаем пустые или незаполненные слоты
		if strings.Contains(device, "Size: No Module Installed") || strings.Contains(device, "Volatile Size: None") {
			continue
		}

		// 1. Извлекаем название слота (SlotLabel)
		locatorReg := regexp.MustCompile(`(?i)Locator:\s*(.+)`)
		slotLabel := "Unknown_Slot"
		if match := locatorReg.FindStringSubmatch(device); len(match) > 1 {
			slotLabel = strings.TrimSpace(match[1])
		}

		// 2. Извлекаем объем памяти и переводим в ГБ
		sizeReg := regexp.MustCompile(`(?i)Size:\s*(\d+)\s*(GB|MB)`)
		var capacityGB uint32 = 0
		if match := sizeReg.FindStringSubmatch(device); len(match) > 2 {
			val, _ := strconv.ParseUint(match[1], 10, 32)
			unit := strings.ToUpper(match[2])
			if unit == "GB" {
				capacityGB = uint32(val)
			} else if unit == "MB" {
				capacityGB = uint32(val / 1024)
				if capacityGB == 0 && val > 0 {
					capacityGB = 1 // Защита от округления
				}
			}
		}

		// Если размер не определился — пропускаем этот логический узел
		if capacityGB == 0 {
			continue
		}

		// 3. Извлекаем производителя
		manufReg := regexp.MustCompile(`(?i)Manufacturer:\s*(.+)`)
		manufacturer := "Unknown"
		if match := manufReg.FindStringSubmatch(device); len(match) > 1 {
			manufacturer = strings.TrimSpace(match[1])
			// Очистка частых "мусорных" ответов DMI
			lowMan := strings.ToLower(manufacturer)
			if manufacturer == "" || strings.Contains(lowMan, "unknown") || strings.Contains(lowMan, "0x") {
				manufacturer = "Unknown"
			}
		}

		// 4. Извлекаем реальную скорость работы шины
		speedReg := regexp.MustCompile(`(?i)Speed:\s*(\d+)\s*MT/s|MHz`)
		var speedMhz uint32 = 0
		if match := speedReg.FindStringSubmatch(device); len(match) > 1 {
			val, _ := strconv.ParseUint(match[1], 10, 32)
			speedMhz = uint32(val)
		}

		// Фолбэк на Configured Clock Speed, если базовое поле пустое
		if speedMhz == 0 {
			confSpeedReg := regexp.MustCompile(`(?i)Configured Clock Speed:\s*(\d+)`)
			if match := confSpeedReg.FindStringSubmatch(device); len(match) > 1 {
				val, _ := strconv.ParseUint(match[1], 10, 32)
				speedMhz = uint32(val)
			}
		}

		// Распределяем глобальные ECC ошибки (или локальные, если они привязаны к хэндлу устройства)
		// В SMBIOS структуры ошибок часто выносятся в отдельные хэндлы, поэтому привязываем к общим
		cErrors := globalCorrectable
		uErrors := globalUncorrectable

		results = append(results, MemorySlotStatus{
			SlotLabel:           slotLabel,
			Manufacturer:        manufacturer,
			CapacityGB:          capacityGB,
			SpeedMhz:            speedMhz,
			CorrectableErrors:   cErrors,
			UncorrectableErrors: uErrors,
		})
	}

	if len(results) > 0 {
		fmt.Printf(" [✅ RAM LOG] Данные по слотам оперативной памяти Unix успешно собраны. Слотов: %d\n", len(results))
		return results
	}

	return getFallbackMemoryStatus()
}

// Резервный фолбэк парсинга, если dmidecode заблокирован контейнеризацией
// (Пытаемся прочитать хотя бы суммарный объем через /proc/meminfo)
func getFallbackMemoryStatus() []MemorySlotStatus {
	cmd := exec.Command("grep", "MemTotal", "/proc/meminfo")
	var out bytes.Buffer
	cmd.Stdout = &out

	var totalGB uint32 = 16 // Дефолтное значение для безопасного ответа
	if cmd.Run() == nil {
		re := regexp.MustCompile(`\d+`)
		if match := re.FindString(out.String()); match != "" {
			kb, _ := strconv.ParseUint(match, 10, 64)
			totalGB = uint32(kb / (1024 * 1024))
		}
	}

	return []MemorySlotStatus{
		{
			SlotLabel:           "SYSTEM_RAM_POOL_0",
			Manufacturer:        "OS_Virtual_Unix_Drive",
			CapacityGB:          totalGB,
			SpeedMhz:            0,
			CorrectableErrors:   0,
			UncorrectableErrors: 0,
		},
	}
}
