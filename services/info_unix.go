//go:build linux || freebsd

package services

import (
	"bytes"
	"fmt"
	"math"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

// HardwareInfo представляет итоговый паспорт машины для HostPulse (DTO сохранено на 100%)
type HardwareInfo struct {
	CpuModel    string `json:"cpu_model"`
	Motherboard string `json:"motherboard"`
	TotalRamGb  uint32 `json:"ram_total_gb"`
	OsType      string `json:"os_type"`
}

// GetHardwareStaticInfo собирает паспортные данные системы (вызывается 1 раз при старте агента)
func GetHardwareStaticInfo() HardwareInfo {
	var info HardwareInfo

	// ----------------------------------------------------
	// 1. ПОЛУЧАЕМ МОДЕЛЬ ПРОЦЕССОРА (CPU)
	// ----------------------------------------------------
	if runtime.GOOS == "linux" {
		if cpuBytes, err := os.ReadFile("/proc/cpuinfo"); err == nil {
			lines := strings.Split(string(cpuBytes), "\n")
			for _, line := range lines {
				if strings.HasPrefix(line, "model name") {
					parts := strings.Split(line, ":")
					if len(parts) >= 2 {
						info.CpuModel = strings.TrimSpace(parts[1])
						break
					}
				}
			}
		}
	} else if runtime.GOOS == "freebsd" {
		cmdCpu := exec.Command("sysctl", "-n", "hw.model")
		var out CpuOut
		cmdCpu.Stdout = &out
		if cmdCpu.Run() == nil {
			info.CpuModel = strings.TrimSpace(out.String())
		}
	}

	if info.CpuModel == "" {
		info.CpuModel = "Unknown Unix CPU"
	}

	// ----------------------------------------------------
	// 2. ПОЛУЧАЕМ ВЕРСИЮ ОПЕРАЦИОННОЙ СИСТЕМЫ (OS)
	// ----------------------------------------------------
	if runtime.GOOS == "linux" {
		// Стандарт для современных дистрибутивов (Ubuntu, CentOS, Debian, Astra)
		if osBytes, err := os.ReadFile("/etc/os-release"); err == nil {
			lines := strings.Split(string(osBytes), "\n")
			var name, version string
			for _, line := range lines {
				if strings.HasPrefix(line, "PRETTY_NAME=") {
					// Если есть PRETTY_NAME, забираем готовое красивое имя в кавычках
					pretty := strings.TrimPrefix(line, "PRETTY_NAME=")
					pretty = strings.Trim(pretty, `"`)
					info.OsType = pretty
					break
				}
				if strings.HasPrefix(line, "NAME=") {
					name = strings.Trim(strings.TrimPrefix(line, "NAME="), `"`)
				}
				if strings.HasPrefix(line, "VERSION=") {
					version = strings.Trim(strings.TrimPrefix(line, "VERSION="), `"`)
				}
			}
			if info.OsType == "" && name != "" {
				info.OsType = fmt.Sprintf("%s %s", name, version)
			}
		}
	} else if runtime.GOOS == "freebsd" {
		cmdUname := exec.Command("uname", "-sr")
		var out bytes.Buffer
		cmdUname.Stdout = &out
		if cmdUname.Run() == nil {
			info.OsType = strings.TrimSpace(out.String())
		}
	}

	if info.OsType == "" {
		info.OsType = "Generic Unix Operating System"
	}

	// ----------------------------------------------------
	// 3. ПОЛУЧАЕМ ОБЩИЙ ОБЪЕМ ОПЕРАТИВНОЙ ПАМЯТИ (RAM)
	// ----------------------------------------------------
	var totalBytes uint64 = 0

	if runtime.GOOS == "linux" {
		if memBytes, err := os.ReadFile("/proc/meminfo"); err == nil {
			lines := strings.Split(string(memBytes), "\n")
			for _, line := range lines {
				if strings.HasPrefix(line, "MemTotal:") {
					// Значение в /proc/meminfo хранится в Килобайтах (kB)
					fields := strings.Fields(line)
					if len(fields) >= 2 {
						if kb, err := strconv.ParseUint(fields[1], 10, 64); err == nil {
							totalBytes = kb * 1024
						}
					}
					break
				}
			}
		}
	} else if runtime.GOOS == "freebsd" {
		cmdMem := exec.Command("sysctl", "-n", "hw.physmem")
		var out bytes.Buffer
		cmdMem.Stdout = &out
		if cmdMem.Run() == nil {
			totalBytes, _ = strconv.ParseUint(strings.TrimSpace(out.String()), 10, 64)
		}
	}

	if totalBytes > 0 {
		gb := float64(totalBytes) / (1024 * 1024 * 1024)
		info.TotalRamGb = uint32(math.Ceil(gb))
	} else {
		info.TotalRamGb = 0
	}

	// ----------------------------------------------------
	// 4. ПОЛУЧАЕМ ДАННЫЕ МАТЕРИНСКОЙ ПЛАТЫ
	// ----------------------------------------------------
	if runtime.GOOS == "linux" {
		vendorBytes, errVendor := os.ReadFile("/sys/class/dmi/id/board_vendor")
		nameBytes, errName := os.ReadFile("/sys/class/dmi/id/board_name")

		if errVendor == nil && errName == nil {
			vendor := strings.TrimSpace(string(vendorBytes))
			name := strings.TrimSpace(string(nameBytes))
			info.Motherboard = fmt.Sprintf("%s %s", vendor, name)
		}
	}

	// Фолбэк для FreeBSD или если sysfs на Linux пуст (например, запуск на виртуалке KVM/Proxmox)
	if info.Motherboard == "" || strings.Contains(strings.ToLower(info.Motherboard), "unknown") {
		// Пробуем извлечь через dmidecode, если есть права суперпользователя
		dmiPath, err := exec.LookPath("dmidecode")
		if err != nil {
			// Проверяем стандартные пути портов FreeBSD и окружения Linux
			commonPaths := []string{
				"/usr/local/sbin/dmidecode", // Основной путь для FreeBSD
				"/usr/sbin/dmidecode",       // Путь для Linux
				"/sbin/dmidecode",
			}
			for _, p := range commonPaths {
				if _, err := os.Stat(p); err == nil {
					dmiPath = p
					break
				}
			}
		}

		// Если утилита найдена в системе, выполняем опрос
		if dmiPath != "" {
			cmdDmi := exec.Command(dmiPath, "-t", "baseboard")
			var out bytes.Buffer
			cmdDmi.Stdout = &out
			if cmdDmi.Run() == nil {
				outputStr := out.String()
				var vendor, product string

				lines := strings.Split(outputStr, "\n")
				for _, line := range lines {
					if strings.Contains(line, "Manufacturer:") {
						vendor = strings.TrimSpace(strings.Split(line, "Manufacturer:")[1])
					}
					if strings.Contains(line, "Product Name:") {
						product = strings.TrimSpace(strings.Split(line, "Product Name:")[1])
					}
				}

				// Защищаем базу от мусорных строк BIOS вроде "To be filled by O.E.M."
				vendorLower := strings.ToLower(vendor)
				productLower := strings.ToLower(product)
				if strings.Contains(vendorLower, "to be filled") || strings.Contains(productLower, "to be filled") {
					vendor = ""
					product = ""
				}

				if vendor != "" || product != "" {
					info.Motherboard = fmt.Sprintf("%s %s", vendor, product)
				}
			}
		}
	}

	if info.Motherboard == "" {
		info.Motherboard = "Unknown Unix Motherboard"
	}

	return info
}

type CpuOut struct {
	bytes.Buffer
}
