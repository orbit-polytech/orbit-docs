# 00. Подготовка рабочего места разработчика

Добро пожаловать в проект **Orbit AI**. Независимо от того, в каком направлении вы будете писать код (Go Backend, AI/Python или Flutter Mobile), микросервисная архитектура проекта требует стандартизированного и воспроизводимого окружения.

В этой главе описан пошаговый процесс настройки рабочей станции: от компиляторов и утилит кодогенерации до Docker-контейнеров и расширений среды разработки.

---

## 1. Системные требования и платформы

* **Поддерживаемые ОС**:
    * **Linux** (Ubuntu 22.04 LTS / 24.04 LTS, Debian, Fedora, Arch);
    * **macOS** (Apple Silicon M1/M2/M3/M4 или Intel, macOS 13+);
    * **Windows 10/11** (строго с установленным **WSL2** и дистрибутивом Ubuntu 22.04/24.04). Разработка бэкенда непосредственно в Windows PowerShell поддерживается, но для работы с Docker и скриптами автоматизации рекомендуется среда Linux/WSL2.
* **Аппаратные ресурсы**:
    * ОЗУ: минимум 8 ГБ (рекомендуется 16 ГБ для комфортного запуска Docker Compose со всеми БД и брокерами);
    * Свободное место на диске: минимум 25 ГБ.

---

## 2. Базовые утилиты и Git

### 2.1. Установка и настройка Git
Git — основа версионирования кода в нашей организации `orbit-polytech`.

=== "Linux (Ubuntu/Debian)"
    ```bash
    sudo apt update && sudo apt install -y git curl wget jq make
    ```

=== "macOS"
    ```bash
    brew install git curl jq make
    ```

=== "Windows (WSL2)"
    Запустите терминал Ubuntu в WSL2:
    ```bash
    sudo apt update && sudo apt install -y git curl wget jq make
    ```

### 2.2. Глобальная конфигурация Git
Установите ваше реальное имя и почту, привязанную к GitHub:

```bash
git config --global user.name "Иван Иванов"
git config --global user.email "ivan.ivanov@example.com"
git config --global init.defaultBranch main
git config --global pull.rebase true
```

Для корректной обработки окончаний строк:
```bash
# Для Linux и macOS:
git config --global core.autocrlf input

# Для Windows (если работаете в PowerShell):
git config --global core.autocrlf true
```

### 2.3. Настройка SSH-ключа для GitHub
Для безопасной работы без постоянного ввода пароля настройте Ed25519-ключ:

```bash
ssh-keygen -t ed25519 -C "ivan.ivanov@example.com"
eval "$(ssh-agent -s)"
ssh-add ~/.ssh/id_ed25519
cat ~/.ssh/id_ed25519.pub
```
Скопируйте полученную строку и добавьте её в свой профиль GitHub: **Settings $\to$ SSH and GPG keys $\to$ New SSH key**.

Проверьте подключение:
```bash
ssh -T git@github.com
# Должно вернуть: Hi username! You've successfully authenticated...
```

---

## 3. Docker и Docker Compose

Все внешние зависимости (PostgreSQL, Redis, RabbitMQ, pgvector) разворачиваются локально в изолированных контейнерах через репозиторий `infra`.

* **Linux**: установите [Docker Engine и Docker Compose v2](https://docs.docker.com/engine/install/ubuntu/) через официальный репозиторий Docker. Обязательно добавьте пользователя в группу `docker`:
    ```bash
    sudo usermod -aG docker $USER
    newgrp docker
    ```
* **macOS / Windows**: установите [Docker Desktop](https://www.docker.com/products/docker-desktop/). На Windows включите интеграцию **Use the WSL 2 based engine**.

Проверка установки:
```bash
docker --version
docker compose version
docker run --rm hello-world
```

---

## 4. Стек Go (Backend трек)

Для микросервисов Orbit используется современный тулчейн Go (версия 1.24+ / 1.26+).

### 4.1. Установка Go
Официальный дистрибутив загружается с [go.dev/dl](https://go.dev/dl/):

=== "Linux / WSL2"
    ```bash
    wget https://go.dev/dl/go1.26.6.linux-amd64.tar.gz
    sudo rm -rf /usr/local/go && sudo tar -C /usr/local -xzf go1.26.6.linux-amd64.tar.gz
    rm go1.26.6.linux-amd64.tar.gz
    ```
    Добавьте пути в `~/.bashrc` (или `~/.zshrc`):
    ```bash
    export PATH=$PATH:/usr/local/go/bin
    export GOPATH=$HOME/go
    export PATH=$PATH:$GOPATH/bin
    ```
    Примените изменения: `source ~/.bashrc`.

=== "macOS"
    ```bash
    brew install go
    ```

### 4.2. Установка линтера `golangci-lint`
Статический анализ кода в Orbit строго автоматизирован:
```bash
# Официальный скрипт установки бинарника golangci-lint:
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin
golangci-lint --version
```

---

## 5. Инструмент работы с контрактами Buf CLI

API-контракты между сервисами описаны на Protocol Buffers v3 и управляются утилитой **Buf** (версия v2):

=== "Linux / WSL2 / macOS"
    ```bash
    # Установка бинарника Buf:
    BIN="/usr/local/bin" && \
    curl -sSL "https://github.com/bufbuild/buf/releases/latest/download/buf-$(uname -s)-$(uname -m)" -o "${BIN}/buf" && \
    chmod +x "${BIN}/buf"
    ```

=== "Windows"
    ```powershell
    # Через Scoop или winget:
    winget install bufbuild.buf
    ```

Проверка:
```bash
buf --version
# Должно вернуть версию 1.30+ или 1.70+
```

---

## 6. Стек Python (AI трек)

Для сервиса искусственного интеллекта (`ai-service`) и Telegram-бота (`bot-service`):

1. **Python 3.12+**:
   ```bash
   sudo apt install -y python3 python3-pip python3-venv
   ```
2. **Менеджер пакетов `uv` (рекомендуется)**:
   ```bash
   curl -LsSf https://astral.sh/uv/install.sh | sh
   ```

---

## 7. Стек Flutter & Dart (Mobile трек)

Для кроссплатформенного клиента `client-mobile`:
1. Установите [Flutter SDK](https://docs.flutter.dev/get-started/install);
2. Добавьте `flutter/bin` в переменную окружения `PATH`;
3. Проверьте окружение командой:
   ```bash
   flutter doctor
   ```

---

## 8. Редактор кода Visual Studio Code

Рекомендуемым редактором для всей команды является **VS Code** (или Cursor).

### Рекомендуемые расширения
Установите через панель расширений (`Ctrl+Shift+X` / `Cmd+Shift+X`) или CLI:

```bash
# Поддержка языков:
code --install-extension golang.Go
code --install-extension ms-python.python
code --install-extension Dart-Code.flutter
code --install-extension bufbuild.vscode-buf

# Инфраструктура и данные:
code --install-extension ms-azuretools.vscode-docker
code --install-extension cweijan.vscode-database-client2

# Совместная работа и Git:
code --install-extension eamodio.gitlens
code --install-extension GitHub.vscode-pull-request-github

# Оформление и качество кода:
code --install-extension esbenp.prettier-vscode
code --install-extension usernamehw.errorlens
```

---

## 9. Комплексная самодиагностика (Smoke Test)

Создайте в домашней директории скрипт `check_env.sh` и запустите его для подтверждения готовности всех компонентов:

```bash
#!/usr/bin/env bash
set -e

echo "=== Проверка окружения Orbit AI ==="
printf "%-15s: " "Git" && git --version
printf "%-15s: " "Docker" && docker --version
printf "%-15s: " "Docker Compose" && docker compose version
printf "%-15s: " "Go" && go version
printf "%-15s: " "golangci-lint" && golangci-lint --version
printf "%-15s: " "Buf CLI" && buf --version
printf "%-15s: " "Python" && python3 --version

echo "======================================"
echo "Все критические инструменты установлены успешно!"
```

Сделайте скрипт исполняемым и выполните:
```bash
chmod +x check_env.sh
./check_env.sh
```

Если все строки вернули версии без ошибок — ваше рабочее место готово. Переходите к следующей главе: **[01. Git и командный workflow с нуля](01-git-basics.md)**.
