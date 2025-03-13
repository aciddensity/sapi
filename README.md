# SAPI

## Running

```bash
go run sapi.go
```

## Building

``` bash
go build sapi.go
```

## Configs

Sapi will look for config options in /etc/sapi.conf.

Default configuration options are as follows:

```INI
logfile="/var/log/sapi.log"
address="0.0.0.0"
port="8080"
```

It is also possible to pass command line arguements to configure Sapi.

Print version and exit

`-v | --version` 

Specify log file location

`-v | --logfile="/var/log/sapi.log"` 

Set listening address

`-a | --address="0.0.0.0"` 

Set listening port

`-p | --port=8080"` 

## Logging

Logs are written to `/var/log/sapi.log` by default.

## RESTful API

The RESTful API endpoints are provided below. You can test these endpoints with curl.

`curl -i localhost:8080/api/v1/version`

### Endpoints
|Description|Path|
|---|---|
|Version|`/api/v1/version`|
|Uptime|`/api/v1/uptime`|
|Disk Usage|`/api/v1/diskusage`|
|OS-Release|`/api/v1/os-release`|

