# a http reverse proxy for tailscale

## build

```shell
go build -v .
```

## config

```json
{
    "auth_key": "tskey-auth-xxxx",
    "servers": [
        {
            "hostname": "jellyfin",
            "url": "http://127.0.0.1:8096"
        }
    ]
}
```

then

```shell
mkdir -p ~/.config/tailscale-reverse-proxy
cp config.json ~/.config/tailscale-reverse-proxy/config.json
```

## run

```shell
./tailscale-reverse-proxy
```

## systemd

```shell
cp tailscale-reverse-proxy ~/.local/bin/
systemctl --user link ${PWD}/tailscale-reverse-proxy.service
systemctl --user start tailscale-reverse-proxy.service
systemctl --user enable tailscale-reverse-proxy.service
```
