# drone-vault-extension

A secret extension that provides optional support for sourcing secrets from Vault. _Please note this project requires Drone server version 1.3 or higher._

## Installation

Create a shared secret:

```bash
$ openssl rand -hex 16
bea26a2221fd8090ea38720fc445eca6
```

Download and run the plugin:

```bash
$ docker run -d \
  --publish=3000:3000 \
  --env=DRONE_DEBUG=true \
  --env=DRONE_SECRET=bea26a2221fd8090ea38720fc445eca6 \
  --env=VAULT_ADDR=... \
  --env=VAULT_TOKEN=... \
  --restart=always \
  --name=drone-vault drone/vault
```

Using approle authentication:

```bash
$ docker run -d \
  --publish=3000:3000 \
  --env=DRONE_DEBUG=true \
  --env=DRONE_SECRET=bea26a2221fd8090ea38720fc445eca6 \
  --env=VAULT_ADDR=... \
  --env=VAULT_AUTH_TYPE=approle \
  --env=VAULT_TOKEN_TTL=72h
  --env=VAULT_TOKEN_RENEWAL=24h
  --env=VAULT_APPROLE_ID=... \
  --env=VAULT_APPROLE_SECRET=... \
  --restart=always \
  --name=drone-vault drone/vault
```

Update your runner configuration to include the plugin address and the shared secret.

```bash
DRONE_SECRET_PLUGIN_ENDPOINT=http://1.2.3.4:3000
DRONE_SECRET_PLUGIN_TOKEN=bea26a2221fd8090ea38720fc445eca6
```
