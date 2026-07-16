data "headscale_external_user" "assignee" {
  name                 = "alice"
  provider_id          = "https://auth.example.com/application/o/headscale/59ac9125-c31b-46c5-814e-06242908cf57"
  create_if_not_exists = true
}
