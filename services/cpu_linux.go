//go:build linux

package services

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type CpuTempStatus struct {
	ZoneName string  `json:"zone_name"`
	Celsius  float64 `json:"celsius"`
}

func SetOhmLibraryBinary(b []byte) {}

func GetCpuTemperatures() []CpuTempStatus {
	var results []CpuTempStatus

	// 1. Попытка через подсистему hwmon (поядерно)
	hwmonPath := "/sys/class/hwmon"
	files, err := os.ReadDir(hwmonPath)
	if err == nil {
		for _, f := range files {
			dirPath := filepath.Join(hwmonPath, f.Name())
			nameBytes, err := os.ReadFile(filepath.Join(dirPath, "name"))
			if err != nil {
				continue
			}
			driverName := strings.TrimSpace(string(nameBytes))

			if driverName == "coretemp" || driverName == "k10temp" || driverName == "zenpower" {
				sensorFiles, err := os.ReadDir(dirPath)
				if err != nil {
					continue
				}
				for _, sf := range sensorFiles {
					sName := sf.Name()
					if strings.HasPrefix(sName, "temp") && strings.HasSuffix(sName, "_input") {
						baseName := strings.TrimSuffix(sName, "_input")
						labelBytes, err := os.ReadFile(filepath.Join(dirPath, baseName+"_label"))
						label := ""
						if err == nil {
							label = strings.TrimSpace(string(labelBytes))
						}
						if label == "" {
							label = fmt.Sprintf("%s_%s", driverName, baseName)
						}

						tempBytes, err := os.ReadFile(filepath.Join(dirPath, sName))
						if err != nil {
							continue
						}
						tempRaw, err := strconv.ParseFloat(strings.TrimSpace(string(tempBytes)), 64)
						if err == nil {
							results = append(results, CpuTempStatus{
								ZoneName: label,
								Celsius:  mathRound(tempRaw/1000.0, 1),
							})
						}
					}
				}
			}
		}
	}

	// 2. Фолбэк на общие зоны ACPI
	if len(results) == 0 {
		thermalPath := "/sys/class/thermal"
		tzFiles, err := os.ReadDir(thermalPath)
		if err == nil {
			for _, tz := range tzFiles {
				if strings.HasPrefix(tz.Name(), "thermal_zone") {
					zoneDir := filepath.Join(thermalPath, tz.Name())
					typeBytes, err := os.ReadFile(filepath.Join(zoneDir, "type"))
					zoneType := "Thermal_Zone"
					if err == nil {
						zoneType = strings.TrimSpace(string(typeBytes))
					}
					tempBytes, err := os.ReadFile(filepath.Join(zoneDir, "temp"))
					if err != nil {
						continue
					}
					tempRaw, err := strconv.ParseFloat(strings.TrimSpace(string(tempBytes)), 64)
					if err == nil {
						results = append(results, CpuTempStatus{
							ZoneName: zoneType,
							Celsius:  mathRound(tempRaw/1000.0, 1),
						})
					}
				}
			}
		}
	}

	if len(results) > 0 {
		return results
	}
	return getFallbackEstimate()
}

func getFallbackEstimate() []CpuTempStatus {
	return []CpuTempStatus{
		{ZoneName: "OS_Hardware_Counters_Unavailable", Celsius: 45.0},
	}
}

func mathRound(val float64, precision int) float64 {
	if precision == 1 {
		return float64(int(val*10+0.5)) / 10
	}
	return float64(int(val+0.5))
}