# xiaomi-exporter

Prometheus exporter for Xiaomi **LYWSD03MMC** temperature/humidity BLE sensors running **original (stock) firmware**.

It periodically scans for the sensors' encrypted BLE advertisements, decrypts them with each device's bind key, and exposes the latest temperature and humidity as Prometheus gauges. Scanning runs in the background on a short duty cycle, so battery drain stays negligible while `/metrics` always serves the most recent values.

## How it works

The exporter talks to the host's BlueZ stack over D-Bus (pure Go, no CGO). It does not need exclusive access to the adapter or HCI passthrough — the host's `bluetoothd` owns the adapter and the exporter just drives discovery.

## Metrics

Exposed at `GET /metrics`:

| Metric                       | Type  | Labels        | Description                 |
| ---------------------------- | ----- | ------------- | --------------------------- |
| `xiaomi_temperature_celsius` | gauge | `mac`, `name` | Last temperature in Celsius |
| `xiaomi_humidity_percent`    | gauge | `mac`, `name` | Last humidity in percent    |

## Configuration

Configured via environment variables (an optional `.env` file in the working directory is loaded automatically).

`DEVICE_NAMES`, `DEVICE_MACS` and `DEVICE_BIND_KEYS` are parallel comma-separated lists — the Nth entry in each describes the same device, so all three must have equal length.

| Variable           | Required | Default | Description                                                   |
| ------------------ | -------- | ------- | ------------------------------------------------------------- |
| `DEVICE_NAMES`     | yes      | —       | Comma-separated device names, used as the `name` metric label |
| `DEVICE_MACS`      | yes      | —       | Comma-separated MAC addresses (e.g. `A4:C1:38:00:00:01`)      |
| `DEVICE_BIND_KEYS` | yes      | —       | Comma-separated bind keys (32 hex chars each)                 |
| `HTTP_ADDR`        | no       | `:8080` | Address the metrics server listens on                         |
| `LOG_LEVEL`        | no       | `info`  | Log level (debug/info/warn/error)                             |
| `SCAN_INTERVAL`    | no       | `5m`    | How often a scan window starts                                |
| `SCAN_WINDOW`      | no       | `90s`   | How long each scan keeps the radio on                         |

### Example `.env`

```dotenv
DEVICE_NAMES="Bedroom,Living Room"
DEVICE_MACS="A4:C1:38:00:00:01,A4:C1:38:00:00:02"
DEVICE_BIND_KEYS="00112233445566778899aabbccddeeff,ffeeddccbbaa99887766554433221100"
```

## Obtaining bind keys

Each sensor's bind key is required to decrypt its advertisements. Extract them from your Xiaomi cloud account with [Xiaomi-cloud-tokens-extractor](https://github.com/PiotrMachowski/Xiaomi-cloud-tokens-extractor) — the tool lists each device's MAC and its bind key.

## Running

### Binary

```bash
make build
DEVICE_NAMES=... DEVICE_MACS=... DEVICE_BIND_KEYS=... ./build/xiaomi-exporter
```

Requires a running `bluetoothd` on the host (`systemctl status bluetooth`).

### Docker

The container speaks D-Bus to the host's BlueZ, so mount the system bus socket.

```bash
docker build -t xiaomi-exporter .
docker run --rm \
  -p 8080:8080 \
  -v /run/dbus/system_bus_socket:/run/dbus/system_bus_socket \
  --env-file .env \
  xiaomi-exporter
```
