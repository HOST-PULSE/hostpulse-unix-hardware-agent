package services

import (
	"bytes"
	"fmt"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

// GetCpuTemperatures опрашивает температурные датчики ядер FreeBSD через sysctl
func GetCpuTemperatures() []CpuTempStatus {
	var results []CpuTempStatus

	// Во FreeBSD температура ядер лежит в ветке dev.cpu
	for i := 0; i < runtime.NumCPU(); i++ {
		cmd := exec.Command("sysctl", "-n", fmt.Sprintf("dev.cpu.%d.temperature", i))
		var out bytes.Buffer
		cmd.Stdout = &out
		if err := cmd.Run(); err == nil {
			raw := strings.TrimSpace(out.String())
			raw = strings.TrimSuffix(raw, "C")
			if val, err := strconv.ParseFloat(raw, 64); err == nil {
				results = append(results, CpuTempStatus{
					ZoneName: fmt.Sprintf("Core %d", i),
					Celsius:  val,
				})
			}
		}
	}

	if len(results) > 0 {
		return results
	}

	return getFallbackEstimate()
}
