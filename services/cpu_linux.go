package services

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// GetCpuTemperatures опрашивает встроенные температурные датчики ядра Linux через sysfs
func GetCpuTemperatures() []CpuTempStatus {
	var results []CpuTempStatus

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
							celsius := tempRaw / 1000.0
							results = append(results, CpuTempStatus{
								ZoneName: label,
								Celsius:  mathRound(celsius, 1),
							})
						}
					}
				}
			}
		}
	}

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
