# deploy

Minimal self-contained deploy folder for running Xray through Docker with a config stored in this directory.

## Files

- `.env.example` - image and docker wrapper settings
- `server.json` - Xray config mounted into the container
- `build_image.sh` - builds the Docker image from the local `xray` binary and saves an image tar
- `start.sh` - validates config in a temporary container and starts Xray with `docker compose`
- `stop.sh` - stops the Docker service
- `status.sh` - shows current container status
- `logs.sh` - tails Docker logs

## Usage

```bash
cd xray-fork/Xray-core/deploy
cp .env.example .env
bash build_image.sh
bash start.sh
```

By default:

- image name is `eiravpn-xray:latest`
- image tar is `./eiravpn-xray-image.tar.gz`
- config directory is `${XRAY_CONFIG_DIR}` and must contain `server.json`
- container uses `network_mode: host`, so Xray listens on the same ports as in the JSON config
