# Docker Deploy

Теперь deploy-развёртка Xray разделена на два независимых профиля:

- `deploy/deploy_allocated`
- `deploy/deploy_shared`

Каждый профиль имеет собственные:

- `configs/server.json`
- `.env.example`
- `docker-compose.yml`
- `bootstrap_server.sh`
- `build_image.sh`
- `start.sh`
- `stop.sh`
- `status.sh`
- `logs.sh`

В корне `deploy/` остаются только общие файлы:

- `build_images.sh`
- `README.md`

`build_image.sh` внутри каждого профиля перед сборкой синхронизирует свой
базовый конфиг из `Xray-core/configs` и сохраняет рядом с профилем образ
`xray`, так что:

- `deploy_allocated` использует `configs/server_allocated.json`
- `deploy_shared` использует `configs/server_shared.json`

## Базовые Конфиги

В исходниках теперь есть два отдельных базовых конфига:

- `xray-fork/Xray-core/configs/server_allocated.json`
- `xray-fork/Xray-core/configs/server_shared.json`

Старый `xray-fork/Xray-core/configs/server.json` оставлен как совместимый
локальный конфиг.

## Как Работать С Профилями

Сейчас сами профили Xray содержат только backend-конфиг. Внешний `:443`
терминируется отдельным `caddy`, который теперь лежит в `agent/deploy` и
роутит по `serverName` на два внутренних REALITY inbound-а Xray. Каждый из них
умеет fallback в локальный `xhttp` inbound.

### Allocated

```bash
cd xray-fork/Xray-core/deploy/deploy_allocated
cp .env.example .env
bash build_image.sh
bash start.sh
```

### Shared

```bash
cd xray-fork/Xray-core/deploy/deploy_shared
cp .env.example .env
bash build_image.sh
bash start.sh
```

## Быстрый Запуск На Новом Сервере

Если на сервер перекинут уже готовый deploy-профиль вместе с
`eiravpn-xray-image.tar.gz`, можно одной командой поставить Docker и сразу
поднять сервис.

### Allocated

```bash
cd xray-fork/Xray-core/deploy/deploy_allocated
cp .env.example .env
bash bootstrap_server.sh
```

### Shared

```bash
cd xray-fork/Xray-core/deploy/deploy_shared
cp .env.example .env
bash bootstrap_server.sh
```

`bootstrap_server.sh` рассчитан на Ubuntu/Debian, ставит Docker Engine +
Compose plugin и затем вызывает `start.sh`.

## Общая Сборка Обоих Образов

Если нужно собрать оба варианта подряд:

```bash
cd xray-fork/Xray-core/deploy
bash build_images.sh
```

Он просто по очереди запускает:

- `deploy_allocated/build_image.sh`
- `deploy_shared/build_image.sh`

## Что Кидать На Сервер

Теперь можно переносить на сервер только один нужный профиль:

- `xray-fork/Xray-core/deploy/deploy_allocated`
- `xray-fork/Xray-core/deploy/deploy_shared`

Обе папки самодостаточны для runtime backend-сервиса: внутри уже есть свои
`common.sh`, `.env.example`, `docker-compose.yml`, `configs/server.json`,
`bootstrap_server.sh` и все управляющие скрипты.

Если образ уже собран локально, достаточно перенести профиль вместе с
`eiravpn-xray-image.tar.gz` и на сервере запускать только:

```bash
cd deploy_allocated
bash start.sh
```

или:

```bash
cd deploy_shared
bash start.sh
```

`build_image.sh` внутри профиля рассчитан на запуск из полного checkout
репозитория. Если на сервере лежит только одна deploy-папка, для запуска нужен
уже готовый tar образа.
