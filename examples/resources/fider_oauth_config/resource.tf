resource "fider_oauth_config" "keycloak" {
  provider      = "keycloak"
  display_name  = "Keycloak"
  status        = 2 # enabled
  client_id     = "fider"
  client_secret = var.keycloak_client_secret

  authorize_url = "https://keycloak.example.com/realms/myrealm/protocol/openid-connect/auth"
  token_url     = "https://keycloak.example.com/realms/myrealm/protocol/openid-connect/token"
  profile_url   = "https://keycloak.example.com/realms/myrealm/protocol/openid-connect/userinfo"
  scope         = "openid profile email"

  is_trusted         = false
  json_user_id_path  = "sub"
  json_user_name_path  = "name"
  json_user_email_path = "email"
}
