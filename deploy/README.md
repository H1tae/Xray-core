# Docker Deploy

Теперь deploy-развёртка Xray разделена на два независимых профиля:

- `deploy/deploy_allocated`
- `deploy/deploy_shared`

Каждый профиль имеет собственные:

- `configs/server.json`
- `.env.example`
- `docker-compose.yml`
- `build_image.sh`
- `start.sh`
- `stop.sh`
- `status.sh`
- `logs.sh`

В корне `deploy/` остаются только общие файлы:

- `build_images.sh`
- `README.md`

`build_image.sh` внутри каждого профиля перед сборкой синхронизирует свой
базовый конфиг из `Xray-core/configs`, так что:

- `deploy_allocated` использует `configs/server_allocated.json`
- `deploy_shared` использует `configs/server_shared.json`

## Базовые Конфиги

В исходниках теперь есть два отдельных базовых конфига:

- `xray-fork/Xray-core/configs/server_allocated.json`
- `xray-fork/Xray-core/configs/server_shared.json`

Старый `xray-fork/Xray-core/configs/server.json` оставлен как совместимый
локальный конфиг.

## Как Работать С Профилями

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

Обе папки самодостаточны для runtime: внутри уже есть свои `common.sh`,
`.env.example`, `docker-compose.yml`, `configs/server.json` и все управляющие
скрипты.

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
