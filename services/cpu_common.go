package services

// CpuTempStatus — общая структура, видимая для poller.go при любой целевой ОС
type CpuTempStatus struct {
	ZoneName string  `json:"zone_name"`
	Celsius  float64 `json:"celsius"`
}

// Утилитарная функция округления, общая для всех платформ
func mathRound(val float64, precision int) float64 {
	if precision == 1 {
		return float64(int(val*10+0.5)) / 10
	}
	return float64(int(val+0.5))
}

// Безопасный откат, если датчики недоступны
func getFallbackEstimate() []CpuTempStatus {
	return []CpuTempStatus{
		{
			ZoneName: "OS_Hardware_Counters_Unavailable",
			Celsius:  45.0,
		},
	}
}
