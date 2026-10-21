//go:build linux || freebsd

package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// AdvancedDiskStatus — Структура полностью сохранена для Django-бэкенда HostPulse
type AdvancedDiskStatus struct {
	DeviceID           string `json:"device_id"`
	Model              string `json:"model"`
	Type               string `json:"type"`
	Interface          string `json:"interface"`
	SerialNumber       string `json:"serial_number"`
	FirmwareVersion    string `json:"firmware_version"`
	HealthStatus       string `json:"health_status"`
	IsHealthy          bool   `json:"is_healthy"`
	HardwareErrors     uint32 `json:"hardware_errors_code"`
	ReallocatedSectors uint32 `json:"reallocated_sectors"`
	BadSectorsPending  uint32 `json:"bad_sectors_pending"`
	CrcErrors          uint32 `json:"crc_errors"`
	PercentageUsed     uint8  `json:"percentage_used"`
	SsdWrittenGb       uint64 `json:"ssd_written_gb"`
	Temperature        uint8  `json:"temperature"`
	PowerOnHours       uint32 `json:"power_on_hours"`
}

type SmartctlScanOutput struct {
	Devices []struct {
		Name     string `json:"name"`     // Например, "/dev/sda" или "/dev/ada0"
		Type     string `json:"type"`     // Например, "ata" или "nvme"
		InfoName string `json:"info_name"`
	} `json:"devices"`
}

type SmartctlOutput struct {
	SerialNumber    string `json:"serial_number"`
	FirmwareVersion string `json:"firmware_version"`
	ModelName       string `json:"model_name"`
	RotationRate int `json:"rotation_rate"`
	SmartStatus struct {
		Passed bool `json:"passed"`
	} `json:"smart_status"`
	PowerOnTime struct {
		Hours uint32 `json:"hours"`
	} `json:"power_on_time"`
	Temperature struct {
		Current uint8 `json:"current"`
	} `json:"temperature"`
	Device struct {
		Type string `json:"type"`
	} `json:"device"`
	AtaSmartAttributes struct {
		Table []struct {
			Id   int    `json:"id"`
			Name string `json:"name"`
			Raw  struct {
				Value uint64 `json:"value"`
			} `json:"raw"`
		} `json:"table"`
	} `json:"ata_smart_attributes"`
	NvmeSmartHealthInformationLog struct {
		Temperature      uint32 `json:"temperature"`
		PowerOnHours     uint32 `json:"power_on_hours"`
		MediaErrors      uint32 `json:"media_errors"`
		PercentageUsed   uint8  `json:"percentage_used"`
		DataUnitsWritten uint64 `json:"data_units_written"`
	} `json:"nvme_smart_health_information_log"`
}

// Пустая функция для сохранения совместимости инициализации в main.go
func SetSmartctlBinary(b []byte) {}

// Ищет нативный smartctl, установленный в операционной системе
func findSmartctlPath() (string, error) {
	if systemPath, err := exec.LookPath("smartctl"); err == nil {
		return systemPath, nil
	}

	// Стандартные пути Unix на случай, если переменная PATH у пользователя root ограничена
	commonPaths := []string{
		"/usr/local/sbin/smartctl", // Стандарт для FreeBSD pkg
		"/usr/sbin/smartctl",       // Стандарт для Linux
		"/sbin/smartctl",
	}
	for _, p := range commonPaths {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}

	return "", fmt.Errorf("утилита smartctl не найдена в системе")
}

// Вспомогательная функция определения SSD для SATA дисков (Разделение Linux / FreeBSD)
func checkIsSataSSD(devicePath string) bool {
	shortName := filepath.Base(devicePath) // "/dev/sda" -> "sda", "/dev/ada0" -> "ada0"

	if runtime.GOOS == "linux" {
		rotationalPath := fmt.Sprintf("/sys/block/%s/queue/rotational", shortName)
		if rotBytes, err := os.ReadFile(rotationalPath); err == nil {
			return strings.TrimSpace(string(rotBytes)) == "0"
		}
	} else if runtime.GOOS == "freebsd" {
		cmd := exec.Command("camcontrol", "identify", shortName)
		var out bytes.Buffer
		cmd.Stdout = &out
		if cmd.Run() == nil {
			return strings.Contains(strings.ToUpper(out.String()), "SOLID STATE")
		}
	}
	return false
}

// GetSmartDataUniversal собирает детальную статистику по конкретному пути диска
func GetSmartDataUniversal(smartctlPath, devicePath string) (res AdvancedDiskStatus, err error) {
	cmd := exec.Command(smartctlPath, "-a", devicePath, "--json")
	output, _ := cmd.Output() // smartctl может возвращать ненулевой код при наличии предупреждений, игнорируем ошибку выполнения

	var data SmartctlOutput
	if err := json.Unmarshal(output, &data); err != nil {
		return res, err
	}

	res.DeviceID = devicePath
	res.Model = data.ModelName
	res.SerialNumber = "Unknown"
	if data.SerialNumber != "" {
		res.SerialNumber = strings.TrimSpace(data.SerialNumber)
	}
	res.FirmwareVersion = "Unknown"
	if data.FirmwareVersion != "" {
		res.FirmwareVersion = strings.TrimSpace(data.FirmwareVersion)
	}

	res.Interface = "Unknown"
	res.Type = "HDD"

	switch strings.ToLower(data.Device.Type) {
	case "ata":
		res.Interface = "SATA"
	case "nvme":
		res.Interface = "NVMe"
		res.Type = "SSD"
	case "scsi":
		res.Interface = "SATA"
	}
	if res.Interface == "Unknown" {
		deviceTypeLower := strings.ToLower(data.Device.Type)

		if data.AtaSmartAttributes.Table != nil ||
			strings.Contains(deviceTypeLower, "sat") ||
			strings.Contains(deviceTypeLower, "scsi") ||
			strings.Contains(deviceTypeLower, "ata") {

			res.Interface = "SATA"
		}
	}
	res.PowerOnHours = data.PowerOnTime.Hours
	res.Temperature = data.Temperature.Current
	res.IsHealthy = data.SmartStatus.Passed
	if res.IsHealthy {
		res.HealthStatus = "Healthy"
	} else {
		res.HealthStatus = "Unhealthy"
	}

	if data.NvmeSmartHealthInformationLog.PowerOnHours > 0 || res.Interface == "NVMe" {
		res.PowerOnHours = data.NvmeSmartHealthInformationLog.PowerOnHours
		res.ReallocatedSectors = data.NvmeSmartHealthInformationLog.MediaErrors
		res.PercentageUsed = data.NvmeSmartHealthInformationLog.PercentageUsed

		if data.NvmeSmartHealthInformationLog.Temperature > 0 {
			res.Temperature = uint8(data.NvmeSmartHealthInformationLog.Temperature)
		}

		if data.NvmeSmartHealthInformationLog.DataUnitsWritten > 0 {
			bytesWritten := data.NvmeSmartHealthInformationLog.DataUnitsWritten * 512000
			res.SsdWrittenGb = bytesWritten / (1024 * 1024 * 1024)
		}
	} else {
		// Парсинг атрибутов для SATA/ATA накопителей
		var lifetimeRemain uint8 = 0
		var wearLevelingCount uint8 = 0
		var blocksWritten uint64 = 0
		var sectorsWritten uint64 = 0

		for _, attr := range data.AtaSmartAttributes.Table {
			switch attr.Id {
			case 5:
				res.ReallocatedSectors = uint32(attr.Raw.Value)
			case 197:
				res.BadSectorsPending = uint32(attr.Raw.Value)
			case 199:
				res.CrcErrors = uint32(attr.Raw.Value)
			case 177: // Wear Leveling Count (Износ/остаток ресурса у многих SSD)
				wearLevelingCount = uint8(attr.Raw.Value)
			case 202: // Percent_Lifetime_Remain (Оставшийся ресурс в % — как на твоем скриншоте)
				lifetimeRemain = uint8(attr.Raw.Value)
			case 241: // Total LBAs Written / Lifetime Writes GiB (Зависит от прошивки)
				blocksWritten = attr.Raw.Value
			case 246: // Total_LBAs_Written (Количество секторов записи — как на твоем скриншоте)
				sectorsWritten = attr.Raw.Value
			}
		}

		// --- ОСТОРОЖНЫЙ И БЕЗОПАСНЫЙ ФИКС ДЛЯ FREEBSD ---
		// Если старая функция checkIsSataSSD вернула true (как на Linux) — отлично.
		// Если она вернула false (как на FreeBSD), мы подстраховываемся анализом названия модели и SMART-атрибутов.
		modelUpper := strings.ToUpper(res.Model)
		isSataSSD := checkIsSataSSD(devicePath) ||
			strings.Contains(modelUpper, "SSD") ||
			strings.Contains(modelUpper, "SOLID STATE") ||
			lifetimeRemain > 0 ||
			wearLevelingCount > 0

		if isSataSSD {
			res.Type = "SSD"

			// 1. Считаем процент износа (PercentageUsed)
			if lifetimeRemain > 0 && lifetimeRemain <= 100 {
				res.PercentageUsed = 100 - lifetimeRemain
			} else if wearLevelingCount > 0 && wearLevelingCount <= 100 {
				res.PercentageUsed = 100 - wearLevelingCount
			}

			// 2. Считаем объем записанных данных (SsdWrittenGb)
			if sectorsWritten > 0 {
				// Переводим LBA-сектора (по 512 байт) в Гигабайты
				res.SsdWrittenGb = (sectorsWritten * 512) / (1024 * 1024 * 1024)
			} else if blocksWritten > 0 {
				// Некоторые контроллеры сразу отдают данные в GB/GiB, проверяем порядок числа
				if blocksWritten > 5000000 { // Скорее всего это тоже сектора
					res.SsdWrittenGb = (blocksWritten * 512) / (1024 * 1024 * 1024)
				} else { // Либо это уже готовые Гигабайты
					res.SsdWrittenGb = blocksWritten
				}
			}
		}
	}

	return res, nil
}


// GetAdvancedDiskStatus — Главная точка входа воркера
func GetAdvancedDiskStatus() []AdvancedDiskStatus {
	fmt.Println(" [💾 DISK SERVICE] Сбор расширенной статистики дисков...")
	var results []AdvancedDiskStatus

	smartctlPath, err := findSmartctlPath()
	if err != nil {
		fmt.Printf(" [❌ DISK ERROR] %v\n", err)
		return results
	}

	cmd := exec.Command(smartctlPath, "--scan", "--json")
	scanOutput, err := cmd.Output()
	if err != nil && len(scanOutput) == 0 {
		fmt.Printf(" [❌ DISK ERROR] Ошибка сканирования шины дисков: %v\n", err)
		return results
	}

	var scanData SmartctlScanOutput
	if err := json.Unmarshal(scanOutput, &scanData); err != nil {
		fmt.Printf(" [❌ DISK ERROR] Ошибка парсинга пула дисков: %v\n", err)
		return results
	}

	for _, dev := range scanData.Devices {
		if dev.Name == "" {
			continue
		}

		diskStatus, err := GetSmartDataUniversal(smartctlPath, dev.Name)
		if err != nil {
			continue
		}

		results = append(results, diskStatus)
	}

	return results
}
