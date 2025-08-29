terraform {
  required_providers {
    system-v1alpha1 = {
      source = "nokia-eda/system-v1alpha1"
      # version = "0.1.0" # Uncomment to specify provider version
    }
  }
}

# Provider configuration
provider "system-v1alpha1" {
  base_url        = ""      # Example: https://eda.mydomain.com:9443 Env var: EDA_BASE_URL
  eda_username    = "admin" # Env var: EDA_USERNAME
  eda_password    = "admin" # Env var: EDA_PASSWORD
  tls_skip_verify = true    # Env var: TLS_SKIP_VERIFY

  # Client secret will be fetched automatically from Keycloak if not set
  # using keycloak credentials
  kc_username       = "admin" # Env var: KC_USERNAME
  kc_password       = "admin" # Env var: KC_PASSWORD
  eda_client_secret = ""      # Env var: EDA_CLIENT_SECRET
}
