#!/bin/bash

# Настройка прерывания скрипта при любой критической ошибке
set -e

# Проверка прав суперпользователя (root)
if [ "$EUID" -ne 0 ] && [ "$(uname)" != "FreeBSD" ]; then
  echo " [❌ ERROR] Пожалуйста, запустите скрипт от имени root (через sudo)."
  exit 1
fi

# Дополнительная проверка прав для FreeBSD
if [ "$(uname)" = "FreeBSD" ] && [ "$(id -u)" -ne 0 ]; then
  echo " [❌ ERROR] Во FreeBSD скрипт должен быть запущен пользователем root."
  exit 1
fi

echo "======================================================="
echo "=== Инсталлятор системной службы HostPulse Agent    ==="
echo "======================================================="

# Константы путей и репозитория (Берутся из твоего скриншота гитхаба)
OS_TYPE="$(uname)"
BINARY_NAME="unix-hardware-agent"
INSTALL_DIR="/usr/local/bin"
GITHUB_REPO="HOST-PULSE/hostpulse-unix-hardware-agent"

# ----------------------------------------------------
# ШАГ 1: Автоматическая установка системных пакетов
# ----------------------------------------------------
echo " [🔄 STEP 1] Проверка и установка системных зависимостей железа..."

if [ "$OS_TYPE" = "FreeBSD" ]; then
    echo " -> Операционная система: FreeBSD"
    export ASSUME_ALWAYS_YES=yes
    if ! type pkg >/dev/null 2>&1; then
        echo " -> Инициализация менеджера пакетов pkg bootstrap..."
        pkg bootstrap -y
    fi
    pkg update
    pkg install -y smartmontools dmidecode

elif [ "$OS_TYPE" = "Linux" ]; then
    echo " -> Операционная система: Linux"
    if [ -f /etc/debian_version ]; then
        # Ubuntu / Debian / Astra Linux
        apt-get update -y
        apt-get install -y smartmontools dmidecode
    elif [ -f /etc/redhat-release ] || [ -f /etc/rocky-release ] || [ -f /etc/almalinux-release ]; then
        # CentOS / RedHat / Rocky / Alma / РЕД ОС
        yum install -y smartmontools dmidecode
    else
        echo " [⚠️ WARNING] Неизвестный дистрибутив Linux. Убедитесь, что smartctl и dmidecode установлены вручную."
    fi
fi

# ----------------------------------------------------
# ШАГ 2: Скачивание нужной архитектуры бинарника из GitHub Releases
# ----------------------------------------------------
echo " [📦 STEP 2] Загрузка скомпилированного Go-бинарника агента..."

# Определяем архитектуру процессора хост-машины
ARCH="$(uname -m)"
if [ "$ARCH" = "x86_64" ]; then
    ARCH_SUFFIX="amd64"
else
    ARCH_SUFFIX="arm64"
fi

# Приводим имя ОС к нижнему регистру (linux или freebsd)
OS_LOWER=$(echo "$OS_TYPE" | tr '[:upper:]' '[:lower:]')

# Формируем имя файла, как оно называется в релизе (например: unix-hardware-agent-linux-amd64)
RELEASE_BINARY_NAME="${BINARY_NAME}-${OS_LOWER}-${ARCH_SUFFIX}"

# Ссылка на скачивание из последнего релиза
DOWNLOAD_URL="https://github.com{GITHUB_REPO}/releases/latest/download/${RELEASE_BINARY_NAME}"

echo " -> Ссылка для скачивания: $DOWNLOAD_URL"

# Скачиваем бинарник напрямую в системную папку, возвращая ему стандартное имя
if curl -sSL -o "$INSTALL_DIR/$BINARY_NAME" "$DOWNLOAD_URL"; then
    chmod 0755 "$INSTALL_DIR/$BINARY_NAME"
    echo " -> Агент успешно скачан и развёрнут в $INSTALL_DIR/$BINARY_NAME"
else
    echo " [❌ ERROR] Не удалось скачать скомпилированный Go-бинарник. Проверьте, что релиз успешно создался в GitHub."
    exit 1
fi

# ----------------------------------------------------
# ШАГ 3: Конфигурация переменных окружения
# ----------------------------------------------------
echo " [⚙️ STEP 3] Настройка переменных окружения службы..."
ENV_DIR="/etc/hostpulse"
ENV_FILE="$ENV_DIR/agent.env"

mkdir -p "$ENV_DIR"

# Перехватываем переменные из однострочной команды, если их нет — ставим безопасные дефолты
if [ -z "$HOSTPULSE_HARDWARE_URL" ]; then
    HOSTPULSE_HARDWARE_URL="https://zedform.kz"
fi

if [ -z "$HOSTPULSE_TOKEN" ]; then
    echo " [⚠️ WARNING] HOSTPULSE_TOKEN не передан. Используется дефолтный токен."
    HOSTPULSE_TOKEN="default_hardware_token_123"
fi

if [ -z "$HOSTPULSE_LOGS_ENABLE" ]; then
    HOSTPULSE_LOGS_ENABLE="true"
fi

# Записываем конфигурационный файл окружения демона
cat <<EOT > "$ENV_FILE"
HOSTPULSE_HARDWARE_URL=${HOSTPULSE_HARDWARE_URL}
HOSTPULSE_TOKEN=${HOSTPULSE_TOKEN}
HOSTPULSE_LOGS_ENABLE=${HOSTPULSE_LOGS_ENABLE}
EOT
chmod 0600 "$ENV_FILE"
echo " -> Конфигурация успешно сохранена в $ENV_FILE"

# ----------------------------------------------------
# ШАГ 4: Регистрация в диспетчере служб (Автозагрузка)
# ----------------------------------------------------
echo " [🛡️ STEP 4] Интеграция службы в автозагрузку системы..."

if [ "$OS_TYPE" = "Linux" ]; then
    # Настройка службы Systemd для Linux
    SYSTEMD_FILE="/etc/systemd/system/hostpulse-hardware.service"

    cat <<EOT > "$SYSTEMD_FILE"
[Unit]
Description=HostPulse Hardware Deep Monitoring Agent
After=network.target

[Service]
Type=simple
EnvironmentFile=$ENV_FILE
ExecStart=$INSTALL_DIR/$BINARY_NAME
Restart=always
RestartSec=10
WorkingDirectory=/tmp

[Install]
WantedBy=multi-user.target
EOT

    systemctl daemon-reload
    systemctl enable hostpulse-hardware.service
    systemctl start hostpulse-hardware.service
    echo " -> Служба Systemd успешно создана, добавлена в автозагрузку и запущена!"

elif [ "$OS_TYPE" = "FreeBSD" ]; then
    # Настройка демона автозагрузки rc.d для FreeBSD
    RC_FILE="/usr/local/etc/rc.d/hostpulse_hardware"

    cat <<'EOT' > "$RC_FILE"
#!/bin/sh
#
# PROVIDE: hostpulse_hardware
# REQUIRE: DAEMON NETWORKING
# KEYWORD: shutdown

. /etc/rc.subr

name="hostpulse_hardware"
rcvar="hostpulse_hardware_enable"

load_rc_config $name

: ${hostpulse_hardware_enable:="NO"}

command="/usr/local/bin/unix-hardware-agent"
pidfile="/var/run/${name}.pid"

start_cmd="hostpulse_hardware_start"

hostpulse_hardware_start() {
    # Экспортируем переменные окружения из нашего конфига перед запуском процесса
    if [ -f "/etc/hostpulse/agent.env" ]; then
        export $(cat /etc/hostpulse/agent.env | grep -v '^#' | xargs)
    fi
    echo "Запуск фонового процесса агента HostPulse..."
    /usr/sbin/daemon -f -p ${pidfile} -t "HostPulse Agent" ${command}
}

run_rc_command "$1"
EOT

    chmod 0555 "$RC_FILE"

    # Автоматически прописываем флаг включения службы в главный системный конфигуратор FreeBSD
    if ! grep -q "hostpulse_hardware_enable" /etc/rc.conf; then
        echo 'hostpulse_hardware_enable="YES"' >> /etc/rc.conf
    fi

    # Запускаем созданную службу
    service hostpulse_hardware start
    echo " -> Служба rc.d успешно добавлена в автозагрузку FreeBSD и запущена в фоне!"
fi

echo "======================================================="
echo " [🎉 SUCCESS] Агент HostPulse успешно развернут как скрытая фоновая служба Unix!"
echo "======================================================="
