# terraform-provider-system-v1alpha1

terraform-provider-system-v1alpha1 is a terraform provider plugin for the `system` resource in Nokia EDA.

## Installation and Usage

Go to <https://docs.eda.dev/latest/development/terraform/>

## Development

To install the provider from the source code, run from the root of the repo:

```bash
make install
```

This make target builds provider plugin binary, stores it under `./build` relative directory and adds the `dev_overrides` block in the `${HOME}/.terraform.rc` file to point to the local binary.

This instructs terraform not to download the provider from the registry and instead look for the provider locally.

To remove the locally built provider run:

```bash
make uninstall
```

This removes the binary from the `./build` directory and removes the corresponding provider key in the `dev_overrides` block from the `${HOME}/.terraform.rc` file.

## Provider configuration variables

| TF variable         | OS env variable     | Default     | Description         |
| ------------------- | ------------------- | ----------- | ------------------- |
| base_url            | EDA_BASE_URL        |             | Base URL            |
| kc_username         | KC_USERNAME         | "admin"     | Keycloak Username   |
| kc_password         | KC_PASSWORD         | "admin"     | Keycloak Password   |
| kc_realm            | KC_REALM            | "master"    | Keycloak Realm      |
| kc_client_id        | KC_CLIENT_ID        | "admin-cli" | Keycloak Client ID  |
| eda_username        | EDA_USERNAME        | "admin"     | EDA Username        |
| eda_password        | EDA_PASSWORD        | "admin"     | EDA Password        |
| eda_realm           | EDA_REALM           | "eda"       | EDA Realm           |
| eda_client_id       | EDA_CLIENT_ID       | "eda"       | EDA Client ID       |
| eda_client_secret   | EDA_CLIENT_SECRET   |             | EDA Client Secret   |
| tls_skip_verify     | TLS_SKIP_VERIFY     | false       | TLS skip verify     |
| rest_debug          | REST_DEBUG          | false       | REST Debug          |
| rest_timeout        | REST_TIMEOUT        | "15s"       | REST Timeout        |
| rest_retries        | REST_RETRIES        | 3           | REST Retries        |
| rest_retry_interval | REST_RETRY_INTERVAL | "5s"        | REST Retry Interval |
