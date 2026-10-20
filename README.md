### HostPulse Unix Hardware Agent 📊

### Choose language / Выберите язык / Тілді таңдаңыз

* 🇷🇺 [Русский](#-русский)
* 🇬🇧 [English](#-english)
* 🇰🇿 [Қазақша](#-қазақша)

---

### 🇷🇺 Русский

Низкоуровневый аппаратный агент для глубокого мониторинга физического состояния серверов на базе Linux (Ubuntu, CentOS, Debian, РЕД ОС и др.) и FreeBSD. Разработан на языке Go с использованием прямых интерфейсов ядра Unix, парсеров виртуальных файловых систем и утилит низкоуровневой инвентаризации. Предназначен для выявления скрытых аппаратных дефектов на серверах автоматизации (iiko, бэк-офисы) до того, как они приведут к критическому сбою бизнес-логики. Работает в фоне как полноценный системный демон (служба Systemd / rc.d).

### ✨ Особенности

* 🦾 **Глубокий аппаратный анализ**: агент не просто считывает загрузку системы, а опрашивает контроллеры дисков, шину ECC-памяти, датчики температур термозон чипсета и резервные линии питания.
* 📦 **Легковесность и нативность**: агент динамически взаимодействует со стабильными системными утилитами (`smartctl`, `dmidecode`), обеспечивая 100% бинарную совместимость с текущей версией ядра операционной системы без раздувания размера файла.
* 🛡 **Отказоустойчивость и изоляция**: методы сбора имеют дублирующие ветки и эвристические фолбэки (чтение `/sys`, `/proc`, вызовы `sysctl`), гарантируя стабильную работу агента как на голом железе (Bare Metal), так и в изолированных виртуальных машинах (Proxmox, KVM, VMware) или Docker-контейнерах.
* 🔒 **Безопасный скрытый режим**: работает полностью в фоне без привязки к управляющему терминалу, абсолютно незаметно для конечного пользователя системы.

### 📊 Собираемые аппаратные метрики

Агент разделяет сбор данных на быстрые и медленные циклы для минимизации нагрузки на хост-машину:

1. **Мониторинг процессора (каждые 5 секунд)**:
    * Поядерный съем реальной температуры из подсистемы `sysfs/hwmon` или ACPI-зон в градусах Цельсия (°C).
    * Автоматический расчет и кэширование пикового нагрева для триггеров защиты от перегрева.
2. **Паспорт и здоровье дисков (каждые 5 минут)**:
    * Извлечение SMART-атрибутов накопителей через атомарные JSON-вызовы: переназначенные сектора (`Reallocated Sectors`), нестабильные сектора (`Bad Sectors Pending`), ошибки кабеля (`CRC Errors`).
    * Автоматическое определение типа диска (SSD/HDD) через маркер вращения шпинделя и полная инвентаризация: серийный номер (`serial_number`) и версия прошивки (`firmware_version`) для быстрой замены диска в стойке.
3. **Паспорт оперативной памяти (каждые 5 минут)**:
    * Покомпонентная структура активных слотов материнской платы (`Locator: DIMM 0`, `DIMM 1`) с отсечением пустых разъемов.
    * Фиксация производителя чипов, частоты работы шины в МГц и чтение аппаратных счетчиков ECC-ошибок (исправляемых и неисправляемых) на уровне контроллера памяти.
4. **Резервирование электропитания (каждые 5 минут)**:
    * Опрос статуса активности блоков питания и сетевых адаптеров через таблицы SMBIOS (Type 39) или интерфейсы питания ядра.
    * Мгновенный алерт при потере фазы, отключении кабеля или аварии одного из БП в резервированной серверной цепи (N+1).
5. **Физическая сеть и линки (каждые 5 минут)**:
    * Контроль физической скорости адаптеров (`speed_mbps`) с аппаратным отсечением виртуальных VPN-хвостов, Docker-мостов и интерфейсов loopback.
    * Фиксация деградации сети (например, если гигабитный порт из-за поврежденного кабеля зажался в аварийные 100 Mbps).
6. **Статический паспорт инвентаризации (1 раз при старте)**:
    * Точная модель CPU, суммарный объем RAM, модель материнской платы и красивое название дистрибутива ОС (через `/etc/os-release` или `uname`).

### 🛠 Быстрый запуск

Агент разворачивается в систему автоматически одной командой. Скрипт инсталляции сам установит необходимые утилиты мониторинга железа, создаст конфигурационные файлы и зарегистрирует службу в автозагрузке.

```bash
curl -sSL https://githubusercontent.com | sudo HOSTPULSE_HARDWARE_URL="{url}" HOSTPULSE_TOKEN="{token}" HOSTPULSE_LOGS_ENABLE="false" bash
```

**Требования к запуску**: строго от имени пользователя **root** (необходимо для чтения сырых таблиц DMI из памяти ядра и взаимодействия с контроллерами дисков через `/dev/`).

### ⚙️ Переменные окружения

* `HOSTPULSE_HARDWARE_URL` — Целевой API эндпоинт вашей CRM для приема аппаратных JSON-пакетов (например, `https://your-domain.com`).
* `HOSTPULSE_TOKEN` — Уникальный мастер-токен авторизации сервера, сгенерированный в панели управления CRM.
---

### 🇬🇧 English

A low-level hardware telemetry agent designed for deep physical health monitoring of servers running Linux (Ubuntu, CentOS, Debian, Rocky, etc.) and FreeBSD. Developed in Go, it interacts directly with Unix kernel structures, sysfs thermal/network trees, and native SMBIOS memory parsers. It is built to detect hidden hardware degradation on automation servers (such as iiko points and back-offices) before it results in critical business downtime. Runs seamlessly as a background system daemon (Systemd service or rc.d script).

### ✨ Features

* 🦾 **Deep Hardware Inspection**: Beyond surface-level OS statistics, the agent queries storage controllers, ECC memory banks, chipset thermal zones, and redundant power lines.
* 📦 **Lightweight & Native Architecture**: Dynamically interfaces with robust system utilities (`smartctl`, `dmidecode`), ensuring 100% binary compatibility with the running kernel version without bloating the file size.
* 🛡 **Fault-Tolerant Fallbacks**: Telemetry routines use multi-layered extraction paths (`/sys`, `/proc` trees, and adaptive `sysctl` fallbacks), ensuring stable operation across Bare Metal, hypervisors (Proxmox, KVM, VMware), and Docker containers.
* 🔒 **Stealth Background Execution**: Completely decoupled from any controlling terminal to run silently as a core system daemon, entirely invisible to users.

### 📊 Collected Hardware Metrics

Telemetry collection is split into high-frequency and low-frequency pollers to ensure near-zero overhead:

1. **CPU Telemetry (Every 5 seconds)**:
   * Per-core temperature reading extracted directly via `sysfs/hwmon` or ACPI thermal zones in Celsius (°C).
   * Real-time calculation of peak thermal layout to trigger critical overheat alert sequences.
2. **Storage SMART & Inventory (Every 5 minutes)**:
   * Extraction of raw SMART attributes via atomic JSON queries: `Reallocated Sectors`, `Bad Sectors Pending`, and interface `CRC Errors`.
   * Automatic SSD/HDD heuristic identification via spindle rotation status and complete inventory profiling (`serial_number` and factory `firmware_version`).
3. **RAM Layout & ECC Health (Every 5 minutes)**:
   * Motherboard physical memory bank slot mapping (`DIMM 0`, `DIMM 1`) omitting empty slots.
   * Logging of chip manufacturer, bus clock speed (MHz), and hardware ECC error correction counters (correctable/uncorrectable) from the memory controller.
4. **Power Supply Redundancy (Every 5 minutes)**:
   * Health, online states, and availability tracking of power units via SMBIOS Power Supply Tables (Type 39) or core power supply subsystems.
   * Real-time alerting upon cable disconnection, power phase drops, or single PSU failures within an N+1 redundant circuit.
5. **Network Interface Link Speed (Every 5 minutes)**:
   * Hard-coded validation of physical link speed (`speed_mbps`) with clean extraction and filtering of virtual VPN/TAP/Docker network adapters.
   * Trapping of link degradation states (e.g., a gigabit interface dropping to 100 Mbps due to a cable crimping fault or broken RJ-45 latch).
6. **Static Inventory Passport (Once at startup)**:
   * Core CPU model, total RAM, motherboard model, and accurate Unix OS distribution string (via `/etc/os-release` or `uname`).

### 🛠 Quick Start

The agent is deployed autonomously using a single-line terminal command. The installation script automatically resolves hardware tool dependencies, handles environment configurations, and registers the background daemon.

```bash
curl -sSL https://githubusercontent.com | sudo HOSTPULSE_HARDWARE_URL="{url}" HOSTPULSE_TOKEN="{token}" HOSTPULSE_LOGS_ENABLE="false" bash
```

**Execution requirement**: Must be executed strictly **As Root** to allow the agent to read low-level SMBIOS/DMI tables from kernel memory and target physical disk devices in `/dev/`.

### ⚙️ Environment Variables

* `HOSTPULSE_HARDWARE_URL` — Target API endpoint of your CRM platform (e.g., `https://your-domain.com`).
* `HOSTPULSE_TOKEN` — Unique UUID authorization token linked to the server asset inside the CRM.

---

### 🇰🇿 Қазақша

Linux (Ubuntu, CentOS, Debian, РЕД ОС және т.б.) және FreeBSD негізіндегі серверлердің физикалық жағдайын терең бақылауға арналған төмен деңгейлі аппараттық агент. Unix ядросының тікелей интерфейстерін, виртуалды файлдық жүйелердің парсерлерін және төмен деңгейлі инвентаризация утилиталарын қолдану арқылы Go тілінде жасалған. Автоматтандыру серверлеріндегі (iiko нүктелері, бэк-офистер) жабдықтың жасырын ақауларын олар бизнес-логиканың критикалық тоқтауына әкеп соқтырғанға дейін анықтауға арналған. Фондық режимде толыққанды жүйелік қызмет (Systemd / rc.d демоны) ретінде жұмыс істейді.

### ✨ Ерекшеліктері

* 🦾 **Терең аппараттық талдау**: агент тек жүйенің жүктелуін оқып қоймайды, сонымен қатар диск контроллерлерін, ECC жады шинасын, чипсеттің термалдык аймақтарын және резервтік қуат беру желілерін тексереді.
* 📦 **Жеңілдік пен нативтілік**: файл өлшемін ұлғайтпай, ағымдағы операциялық жүйе ядросының нұсқасымен 100% бинарлық үйлесімділікті қамтамасыз ете отырып, тұрақты жүйелік утилиталармен (`smartctl`, `dmidecode`) динамикалық түрде әрекеттеседі.
* 🛡 **Ақауларға төзімділік пен оқшаулау**: жинау әдістерінде дубликатталатын тармақтар мен эвристикалық фолбэктер бар (`/sys`, `/proc` оқу, `sysctl` шақырулары), бұл агенттің голы темірде (Bare Metal), виртуалды машиналарда (Proxmox, KVM, VMware) немесе Docker контейнерлерінде тұрақты жұмысына кепілдік береді.
* 🔒 **Қауіпсіз жасырын режим**: басқару терминалына байланбай, фондық режимде толықтай жұмыс істейді, жүйенің соңғы пайдаланушысына мүлдем білінбейді.

### 📊 Жиналатын аппараттық метрикалар

Агент хост-машинаға түсетін жүктемені азайту үшін деректерді жинауды жылдам және баяу циклдарға бөледі:

1. **Процессор мониторингі (әр 5 секунд сайын)**:
   * Жүйелік `sysfs/hwmon` немесе ACPI аймақтарынан Цельсий градусымен (°C) нақты температураны ядро бойынша алу.
   * Қызып кетуден қорғау триггерлері үшін ең жоғары қызуды автоматты түрде есептеу және кэштеу.
2. **Дисткилердің паспорты мен денсаулығы (әр 5 минут сайын)**:
   * Атомарлық JSON шақырулары арқылы жинақтауыштардың SMART атрибуттарын алу: қайта тағайындалған секторлар (`Reallocated Sectors`), тұрақсыз секторлар (`Bad Sectors Pending`), кабель қателіктері (`CRC Errors`).
   * Шпиндельдің айналу маркері арқылы дисктің түрін (SSD/HDD) автоматты түрде анықтау және толық инвентаризация: сервер сөресіндегі ақаулы дискті жылдам ауыстыру үшін сериялық нөмір (`serial_number`) мен микробағдарлама нұсқасы (`firmware_version`).
3. **Жедел жадтың паспорты (әр 5 минут сайын)**:
   * Бос ұяшықтарды кесіп тастап, аналық тақтаның белсенді слоттарының құрылымы (`Locator: DIMM 0`, `DIMM 1`).
   * Чип өндірушісін, МГц-тегі шина жиілігін бекіту және жад контроллері деңгейінде ECC қателіктерінің (түзетілетін және түзетілмейтін) аппараттық есептегіштерін оқу.
4. **Қуат көздерін резервтеу (әр 5 минут сайын)**:
   * SMBIOS кестелері (Type 39) немесе ядроның қуат интерфейстері арқылы қуат беру блоктары мен желілік адаптерлердің белсенділік күйін тексеру.
   * Резервтелген серверлік тізбекте (N+1) фаза жоғалғанда, кабель ажыратылғанда немесе БП-ның бірі істен шыққанда лезде алерт беру.
5. **Физикалық желі және линктер (әр 5 минут сайын)**:
   * Виртуалды VPN, Docker көпірлерін және loopback интерфейстерін аппараттық түрде кесіп тастап, адаптерлердің нақты физикалық жылдамдығын (`speed_mbps`) бақылау.
   * Желінің нашарлауын бекіту (мысалы, егер гигабиттік порт зақымдалған кабель салдарынан авариялық 100 Mbps жылдамдыққа түсіп қалса).
6. **Статикалық инвентаризация паспорты (іске қосылғанда 1 рет)**:
   * Нақты CPU моделі, жалпы RAM көлемі, аналық тақта моделі және ОС дистрибутивінің атауы (`/etc/os-release` немесе `uname` арқылы).

### 🛠 Жылдам іске қосу

Агент жүйеге бір командамен автоматты түрде орнатылады. Орнату скрипті қажетті жабдықты бақылау утилиталарын өзі орнатады, конфигурациялық файлдарды жасайды және қызметті автоіске қосуға тіркейді.

```bash
curl -sSL https://githubusercontent.com | sudo HOSTPULSE_HARDWARE_URL="{url}" HOSTPULSE_TOKEN="{token}" HOSTPULSE_LOGS_ENABLE="false" bash
```

**Іске қосу талаптары**: тек **root** пайдаланушысы атынан (ядро жадынан DMI кестелерін оқу және `/dev/` ішіндегі физикалық дискілермен әрекеттесу үшін қажет).

### ⚙️ Переменные окружения / Environment Variables

* `HOSTPULSE_HARDWARE_URL` — Целевой API эндпоинт вашей CRM для приема аппаратных JSON-пакетов (например, `https://your-domain.com`).
* `HOSTPULSE_TOKEN` — Уникальный мастер-токен авторизации сервера, сгенерированный в панели управления CRM.



