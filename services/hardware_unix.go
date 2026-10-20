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

var smartctlBinary []byte

// SetSmartctlBinary принимает массив байт зашитого smartctl из main.go
func SetSmartctlBinary(b []byte) {
	smartctlBinary = b
}

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
		Name     string `json:"name"`     // например, "/dev/sda" или "/dev/ada0"
		Type     string `json:"type"`     // например, "ata" или "nvme"
		InfoName string `json:"info_name"`
	} `json:"devices"`
}

type SmartctlOutput struct {
	SerialNumber    string `json:"serial_number"`
	FirmwareVersion string `json:"firmware_version"`
	ModelName       string `json:"model_name"`

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

// extractSmartctl извлекает зашитый smartctl во временную папку ОС
func extractSmartctl() (string, error) {
	tmpDir := os.TempDir()
	execPath := filepath.Join(tmpDir, "hostpulse_smartctl")

	if len(smartctlBinary) == 0 {
		return "", fmt.Errorf("массив smartctlBinary пуст! Проверьте //go:embed")
	}

	if _, err := os.Stat(execPath); err == nil {
		_ = os.Remove(execPath)
	}

	err := os.WriteFile(execPath, smartctlBinary, 0755)
	if err != nil {
		return "", err
	}

	return execPath, nil
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
		// В FreeBSD опрашиваем camcontrol на предмет наличия флага SOLID STATE
		cmd := exec.Command("camcontrol", "identify", shortName)
		var out bytes.Buffer
		cmd.Stdout = &out
		if err := cmd.Run(); err == nil {
			return strings.Contains(strings.ToUpper(out.String()), "SOLID STATE")
		}
	}
	return false
}

// GetSmartDataUniversal собирает детальную статистику по конкретному пути диска
func GetSmartDataUniversal(smartctlPath, devicePath string) (res AdvancedDiskStatus, err error) {
	cmd := exec.Command(smartctlPath, "-a", devicePath, "--json")
	output, _ := cmd.Output()

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
		for _, attr := range data.AtaSmartAttributes.Table {
			switch attr.Id {
			case 5:
				res.ReallocatedSectors = uint32(attr.Raw.Value)
			case 197:
				res.BadSectorsPending = uint32(attr.Raw.Value)
			case 199:
				res.CrcErrors = uint32(attr.Raw.Value)
			}
		}

		// Запускаем кроссплатформенную проверку на SSD
		if checkIsSataSSD(devicePath) {
			res.Type = "SSD"
		}
	}

	return res, nil
}

func GetAdvancedDiskStatus() []AdvancedDiskStatus {
	fmt.Println(" [💾 DISK SERVICE] Сбор расширенной статистики дисков...")
	var results []AdvancedDiskStatus

	smartctlPath, err := extractSmartctl()
	if err != nil {
		fmt.Printf(" [❌ DISK ERROR] Не удалось распаковать утилиту smartctl: %v\n", err)
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
