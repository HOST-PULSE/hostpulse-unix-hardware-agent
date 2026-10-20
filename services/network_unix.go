//go:build linux || freebsd

package services

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

// NetworkInterfaceStatus — Структура полностью сохранена для Django-бэкенда HostPulse
type NetworkInterfaceStatus struct {
	InterfaceName string `json:"interface_name"` // Название адаптера (например: eth0, enp3s0)
	MACAddress    string `json:"mac_address"`    // MAC-адрес для точной идентификации карты
	SpeedMbps     uint64 `json:"speed_mbps"`     // Текущая скорость линка в Mbps (например: 1000, 100)
	IsConnected   bool   `json:"is_connected"`   // Подключен ли кабель физически
}

// GetNetworkHardwareStatus собирает статус по всем активным физическим сетевым интерфейсам в Unix
func GetNetworkHardwareStatus() []NetworkInterfaceStatus {
	var results []NetworkInterfaceStatus

	// Шаг 1: Получаем список всех интерфейсов и их MAC-адресов стандартными средствами Go
	goInterfaces, err := net.Interfaces()
	if err != nil {
		fmt.Printf(" [❌ NET ERROR] Не удалось получить сетевые интерфейсы ОС: %v\n", err)
		return nil
	}

	for _, iface := range goInterfaces {
		name := iface.Name

		// Шаг 2: Фильтруем виртуальный мусор (Loopback, Docker, VPN, TUN/TAP)
		// В Linux у любого физического адаптера в sysfs есть поддиректория "device"
		if runtime.GOOS == "linux" {
			devicePath := filepath.Join("/sys/class/net", name, "device")
			if _, err := os.Stat(devicePath); os.IsNotExist(err) {
				continue // Пропускаем, это виртуальный интерфейс (VPN, Docker, lo)
			}
		} else if runtime.GOOS == "freebsd" {
			// Во FreeBSD физические интерфейсы обычно начинаются на em, igb, ix, re, bge, alc
			// Отсекаем явный локальный и виртуальный мусор
			if name == "lo0" || strings.HasPrefix(name, "tun") || strings.HasPrefix(name, "tap") || strings.HasPrefix(name, "pflog") {
				continue
			}
		}

		mac := iface.HardwareAddr.String()
		if mac == "" {
			continue // Физическая карта обязана иметь MAC-адрес
		}

		// Шаг 3: Собираем статус подключения и скорость линка
		var isConnected bool = false
		var speedMbps uint64 = 0

		if runtime.GOOS == "linux" {
			// Читаем операционный статус интерфейса (up / down)
			operstateBytes, err := os.ReadFile(filepath.Join("/sys/class/net", name, "operstate"))
			if err == nil {
				state := strings.TrimSpace(string(operstateBytes))
				// up или unknown (некоторые старые карты или специфичные драйверы пишут unknown при активном линке)
				if state == "up" || state == "unknown" {
					isConnected = true
				}
			}

			// Если кабель физически подключен, забираем скорость линка
			if isConnected {
				speedBytes, err := os.ReadFile(filepath.Join("/sys/class/net", name, "speed"))
				if err == nil {
					speedStr := strings.TrimSpace(string(speedBytes))
					if parsedSpeed, err := strconv.ParseUint(speedStr, 10, 64); err == nil {
						// Ядро Linux возвращает скорость сразу в Mbps (например: 100, 1000)
						// Защита от ошибочных системных кодов или состояния сна карты (-1 в uint64)
						if parsedSpeed > 0 && parsedSpeed < 1000000 {
							speedMbps = parsedSpeed
						}
					}
				}
			}
		} else if runtime.GOOS == "freebsd" {
			// Во FreeBSD статус и скорость извлекаются через проверку флагов интерфейса
			if (iface.Flags & net.FlagUp) != 0 {
				isConnected = true
				// По умолчанию для FreeBSD выставляем стандартный гигабит,
				// полноценный парсинг медиа-типа требует вызова утилиты ifconfig <name>
				speedMbps = 1000
			}
		}

		// Формируем итоговый объект для отправки на бэкэнд
		results = append(results, NetworkInterfaceStatus{
			InterfaceName: name,
			MACAddress:    strings.ToUpper(mac), // Приводим к верхнему регистру для единообразия с Windows
			SpeedMbps:     speedMbps,
			IsConnected:   isConnected,
		})
	}

	if len(results) > 0 {
		fmt.Printf(" [✅ NET LOG] Данные по сетевым интерфейсам собраны. Найдено адаптеров: %d\n", len(results))
	}
	return results
}
