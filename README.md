# drone-vault-extension

A secret extension that provides optional support for sourcing secrets from Vault. _Please note this project requires Drone server version 1.3 or higher._

## Installation

Create a shared secret:

```text
$ openssl rand -hex 16
bea26a2221fd8090ea38720fc445eca6
```

Download and run the plugin:

```text
$ docker run -d \
  --publish=3000:3000 \
  --env=DRONE_DEBUG=true \
  --env=DRONE_SECRET=bea26a2221fd8090ea38720fc445eca6 \
  --env=VAULT_ADDR=... \
  --env=VAULT_TOKEN=... \
  --restart=always \
  --name=drone-vault drone/vault
```

Update your Drone agent configuration to include the plugin address and the shared secret.

```text
DRONE_SECRET_ENDPOINT=http://1.2.3.4:3000
DRONE_SECRET_SECRET=bea26a2221fd8090ea38720fc445eca6
```

You can configure the plugin with the following DRONE environment variables:

```text
string        DRONE_BIND
bool          DRONE_DEBUG
string        DRONE_SECRET
string        VAULT_ADDR
time.Duration VAULT_TOKEN_RENEWAL
time.Duration VAULT_TOKEN_TTL
string        VAULT_AUTH_TYPE
string        VAULT_AUTH_MOUNT_POINT
string        VAULT_KUBERNETES_ROLE
```

For example, if you'd like to change the port the plugin serves, use:
```text
docker run --publish=3001:3001 --env=DRONE_BIND=0.0.0.0:3001 ...
```

This plugin accepts the following [VAULT environment variables](https://www.vaultproject.io/docs/commands/index.html#environment-variables) for the vault client:

```text
VAULT_ADDR
VAULT_CACERT
VAULT_CAPATH
VAULT_CLIENT_CERT
VAULT_SKIP_VERIFY
VAULT_MAX_RETRIES
VAULT_TOKEN
VAULT_TLS_SERVER_NAME
```

One use-case for these env vars would be if you secured your vault endpoint with TLS and a self-signed certificate.
You could then insert the CA into the drone-vault plugin container like this (considered you've copied the `ca.crt` file to the host):

```text
docker run -d \
-v /home/ubuntu/ca.crt:/ca.crt \
--publish=3001:3001 \
--env=DRONE_BIND=0.0.0.0:3001 \
--env=DRONE_DEBUG=true \
--env=DRONE_SECRET=${DRONE_SECRET} \
--env=VAULT_CACERT=/ca.crt \
--env=VAULT_ADDR=https://${VAULT_IP_OR_HOSTNAME}:8200 \
--env=VAULT_TOKEN=${VAULT_TOKEN} \
--restart=always \
--name=drone-vault drone/vault
```
