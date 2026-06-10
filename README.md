# terraform-provider-fider

Terraform / OpenTofu provider for managing [Fider](https://fider.io) self-hosted feedback platform configuration.

Published on the [Terraform Registry](https://registry.terraform.io/providers/m11s-io/fider).

## Requirements

- Terraform >= 1.0 or OpenTofu >= 1.6
- Fider instance with an admin user API key

## Usage

```hcl
terraform {
  required_providers {
    fider = {
      source  = "m11s-io/fider"
      version = "~> 0.1"
    }
  }
}

provider "fider" {
  url     = "https://feedback.example.com"
  api_key = var.fider_api_key
}

resource "fider_oauth_config" "keycloak" {
  provider_name = "keycloak"
  display_name  = "Keycloak"
  client_id     = "fider"
  client_secret = var.keycloak_client_secret

  authorize_url = "https://keycloak.example.com/realms/myrealm/protocol/openid-connect/auth"
  token_url     = "https://keycloak.example.com/realms/myrealm/protocol/openid-connect/token"
  profile_url   = "https://keycloak.example.com/realms/myrealm/protocol/openid-connect/userinfo"
  scope         = "openid profile email"

  json_user_id_path    = "sub"
  json_user_name_path  = "name"
  json_user_email_path = "email"
}
```

## Resources

| Resource | Description |
|---|---|
| `fider_oauth_config` | Custom OAuth2 / OIDC provider for a Fider tenant |

## Notes

- Each Fider tenant requires its own provider block with that tenant's URL and API key. API keys are scoped per tenant.
- Fider has no delete API for custom OAuth configs. On `terraform destroy`, the resource is disabled rather than removed.
- Admin API keys are generated in Fider user settings → **API Key**.

## Development

```bash
make build    # compile
make install  # install locally for testing
make test     # run tests
```

## License

MIT
