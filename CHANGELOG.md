# Changelog

## [1.1.1](https://github.com/rixlhq/terraform-provider-netcup/compare/v1.1.0...v1.1.1) (2026-08-14)


### Bug Fixes

* **release:** trigger goreleaser on release:published and workflow_dispatch ([42a48ad](https://github.com/rixlhq/terraform-provider-netcup/commit/42a48ad47a27aebbf33e04cd7a81c58284efadaa))

## [1.1.0](https://github.com/rixlhq/terraform-provider-netcup/compare/v1.0.0...v1.1.0) (2026-08-14)


### Features

* **client:** implement netcup DNS API JSON client ([429c84f](https://github.com/rixlhq/terraform-provider-netcup/commit/429c84fe8fde7fdc9a5c2f8f12caf7556ec9e50a))
* **provider:** add ip_version function and scp_access_token ephemeral resource ([b8dd555](https://github.com/rixlhq/terraform-provider-netcup/commit/b8dd555e63bc62b5ac63e02abc3884280e573b9e))
* **provider:** add netcup provider, dns record and zone resources ([22b9bd2](https://github.com/rixlhq/terraform-provider-netcup/commit/22b9bd2fecad454bd28ee622060a7a320b04a427))
* **provider:** add netcup_scp_server_interface resource and task result handling ([8e500bf](https://github.com/rixlhq/terraform-provider-netcup/commit/8e500bf728b2548e3ab740f811d5c45c4a7718d7))
* **provider:** support environment variables for provider credentials and URLs ([93f78c2](https://github.com/rixlhq/terraform-provider-netcup/commit/93f78c2fada58280cb1231b5e88e23848c8fc37f))
* **scp:** add IPv4/IPv6 failover IP and user VLAN resources, extend generic CRUD for list reads and no-delete ([bbd696f](https://github.com/rixlhq/terraform-provider-netcup/commit/bbd696fbc36e6f24aca2ee6c59f623ec6c3f260d))
* **scp:** add netcup_scp_rdns resource and make generic CRUD base more flexible ([9cfbef6](https://github.com/rixlhq/terraform-provider-netcup/commit/9cfbef68d2cbb122b9d9cc51f7a1db13f45cdc3e))
* **scp:** add netcup_scp_server resource for managing server attributes ([806ae96](https://github.com/rixlhq/terraform-provider-netcup/commit/806ae966216ee329249b48ffa088bc14d47c91a4))
* **scp:** add netcup_scp_server_action resource for one-off server actions ([90f8f83](https://github.com/rixlhq/terraform-provider-netcup/commit/90f8f83e52afe9aaa6de33ad470fc792f4b44f75))
* **scp:** add netcup_scp_server_interface_firewall resource ([24d56e4](https://github.com/rixlhq/terraform-provider-netcup/commit/24d56e481d2fb928cce733a79de29f0cf47accd0))
* **scp:** add netcup_scp_server_snapshot resource ([aac511a](https://github.com/rixlhq/terraform-provider-netcup/commit/aac511af610916f1f1e58a3379a2a448fdb0a7d5))
* **scp:** add SCP REST API client and shared provider data ([b6f106f](https://github.com/rixlhq/terraform-provider-netcup/commit/b6f106f4ddfd76d445470c381372a136fa9a0c1b))
* **scp:** add server metrics data source, disk driver update action, and task cancel action ([0035ca7](https://github.com/rixlhq/terraform-provider-netcup/commit/0035ca7c2f270c8b9103655d3292ea56dfd24883))
* **scp:** add terraform import support to crud resources ([6cdfb82](https://github.com/rixlhq/terraform-provider-netcup/commit/6cdfb82b5302c2a2b1070e5f0a56be30958f3eba))
* **scp:** add user and user SSH key resources; update README and examples ([0b831a7](https://github.com/rixlhq/terraform-provider-netcup/commit/0b831a7b1e948d41dcd8a8eabf049531d25b2a25))
* **scpclient:** add retries, task polling and structured logging ([2267de9](https://github.com/rixlhq/terraform-provider-netcup/commit/2267de9c40daca4d269e9c2626a1caaab80a313e))
* **scp:** generate data sources from SCP OpenAPI specification ([a5b6544](https://github.com/rixlhq/terraform-provider-netcup/commit/a5b6544a661af579d9f668e85e42dfd7b5bd5d62))
* **scp:** generic CRUD resource and netcup_scp_user_firewall_policy ([5457902](https://github.com/rixlhq/terraform-provider-netcup/commit/545790287d972fdc1e65a9392b3366c3e7fc3002))
* Terraform provider for Netcup CCP DNS and SCP REST API ([8539a46](https://github.com/rixlhq/terraform-provider-netcup/commit/8539a463798e555a3c6a8cf203176beb666e0212))


### Bug Fixes

* **client:** align test request field with apipassword json tag and prefer errors.New ([99be9eb](https://github.com/rixlhq/terraform-provider-netcup/commit/99be9eb1861aa5bf3f196092bf69250f2ca46b16))
* **provider:** accept SCP refresh-token-only configuration ([13ef601](https://github.com/rixlhq/terraform-provider-netcup/commit/13ef6016e21d0c86c7de113ac620eca779dc0617))
* **provider:** add schema descriptions and set goreleaser project_name ([b5fc5d6](https://github.com/rixlhq/terraform-provider-netcup/commit/b5fc5d6cfda99f76051569a83646cd91855e3499))
* **scp:** add createReadsBack for task-style create responses and clean up readState helper ([d20faac](https://github.com/rixlhq/terraform-provider-netcup/commit/d20faacf056a90f4cc1d870ebaf44210577ca6b4))
* **scp:** correct path parameter order and user_id substitution in generated data sources ([0e57f3c](https://github.com/rixlhq/terraform-provider-netcup/commit/0e57f3c5788df16eea1a5a3b119c674104440eaf))
* **scp:** generic CRUD path, body key mapping, and response root fallback ([ac9b1c4](https://github.com/rixlhq/terraform-provider-netcup/commit/ac9b1c438514781e0327efe5087b2d0a1b19867c))
* **scp:** guard nil queryBuilder in scp_server_action resource ([66cb843](https://github.com/rixlhq/terraform-provider-netcup/commit/66cb8436bff6b7d0fa1ff99ffd203d122eb8b983))
* **scp:** normalize camelCase JSON keys to snake_case and set correct POST content-type ([bc07f89](https://github.com/rixlhq/terraform-provider-netcup/commit/bc07f89c46f27dc72c54e38ebf27e5f5ce20bff2))
* **scp:** preserve known path parameters and config values in generic CRUD state ([8dd5a2e](https://github.com/rixlhq/terraform-provider-netcup/commit/8dd5a2e618c5afb24a833d2d689a97eb2e78682f))
* **scp:** route generic CRUD Read through readState to honor readFromList ([9b2c3d2](https://github.com/rixlhq/terraform-provider-netcup/commit/9b2c3d29a94b3a92ee61c87f3b2f105c1ee27eaa))

## 1.0.0 (2026-08-08)


### Features

* **client:** implement netcup DNS API JSON client ([429c84f](https://github.com/rixlhq/terraform-provider-netcup/commit/429c84fe8fde7fdc9a5c2f8f12caf7556ec9e50a))
* **provider:** add ip_version function and scp_access_token ephemeral resource ([b8dd555](https://github.com/rixlhq/terraform-provider-netcup/commit/b8dd555e63bc62b5ac63e02abc3884280e573b9e))
* **provider:** add netcup provider, dns record and zone resources ([22b9bd2](https://github.com/rixlhq/terraform-provider-netcup/commit/22b9bd2fecad454bd28ee622060a7a320b04a427))
* **provider:** add netcup_scp_server_interface resource and task result handling ([8e500bf](https://github.com/rixlhq/terraform-provider-netcup/commit/8e500bf728b2548e3ab740f811d5c45c4a7718d7))
* **provider:** support environment variables for provider credentials and URLs ([93f78c2](https://github.com/rixlhq/terraform-provider-netcup/commit/93f78c2fada58280cb1231b5e88e23848c8fc37f))
* **scp:** add IPv4/IPv6 failover IP and user VLAN resources, extend generic CRUD for list reads and no-delete ([bbd696f](https://github.com/rixlhq/terraform-provider-netcup/commit/bbd696fbc36e6f24aca2ee6c59f623ec6c3f260d))
* **scp:** add netcup_scp_rdns resource and make generic CRUD base more flexible ([9cfbef6](https://github.com/rixlhq/terraform-provider-netcup/commit/9cfbef68d2cbb122b9d9cc51f7a1db13f45cdc3e))
* **scp:** add netcup_scp_server resource for managing server attributes ([806ae96](https://github.com/rixlhq/terraform-provider-netcup/commit/806ae966216ee329249b48ffa088bc14d47c91a4))
* **scp:** add netcup_scp_server_action resource for one-off server actions ([90f8f83](https://github.com/rixlhq/terraform-provider-netcup/commit/90f8f83e52afe9aaa6de33ad470fc792f4b44f75))
* **scp:** add netcup_scp_server_interface_firewall resource ([24d56e4](https://github.com/rixlhq/terraform-provider-netcup/commit/24d56e481d2fb928cce733a79de29f0cf47accd0))
* **scp:** add netcup_scp_server_snapshot resource ([aac511a](https://github.com/rixlhq/terraform-provider-netcup/commit/aac511af610916f1f1e58a3379a2a448fdb0a7d5))
* **scp:** add SCP REST API client and shared provider data ([b6f106f](https://github.com/rixlhq/terraform-provider-netcup/commit/b6f106f4ddfd76d445470c381372a136fa9a0c1b))
* **scp:** add server metrics data source, disk driver update action, and task cancel action ([0035ca7](https://github.com/rixlhq/terraform-provider-netcup/commit/0035ca7c2f270c8b9103655d3292ea56dfd24883))
* **scp:** add terraform import support to crud resources ([6cdfb82](https://github.com/rixlhq/terraform-provider-netcup/commit/6cdfb82b5302c2a2b1070e5f0a56be30958f3eba))
* **scp:** add user and user SSH key resources; update README and examples ([0b831a7](https://github.com/rixlhq/terraform-provider-netcup/commit/0b831a7b1e948d41dcd8a8eabf049531d25b2a25))
* **scpclient:** add retries, task polling and structured logging ([2267de9](https://github.com/rixlhq/terraform-provider-netcup/commit/2267de9c40daca4d269e9c2626a1caaab80a313e))
* **scp:** generate data sources from SCP OpenAPI specification ([a5b6544](https://github.com/rixlhq/terraform-provider-netcup/commit/a5b6544a661af579d9f668e85e42dfd7b5bd5d62))
* **scp:** generic CRUD resource and netcup_scp_user_firewall_policy ([5457902](https://github.com/rixlhq/terraform-provider-netcup/commit/545790287d972fdc1e65a9392b3366c3e7fc3002))
* Terraform provider for Netcup CCP DNS and SCP REST API ([8539a46](https://github.com/rixlhq/terraform-provider-netcup/commit/8539a463798e555a3c6a8cf203176beb666e0212))


### Bug Fixes

* **client:** align test request field with apipassword json tag and prefer errors.New ([99be9eb](https://github.com/rixlhq/terraform-provider-netcup/commit/99be9eb1861aa5bf3f196092bf69250f2ca46b16))
* **provider:** accept SCP refresh-token-only configuration ([13ef601](https://github.com/rixlhq/terraform-provider-netcup/commit/13ef6016e21d0c86c7de113ac620eca779dc0617))
* **provider:** add schema descriptions and set goreleaser project_name ([b5fc5d6](https://github.com/rixlhq/terraform-provider-netcup/commit/b5fc5d6cfda99f76051569a83646cd91855e3499))
* **scp:** add createReadsBack for task-style create responses and clean up readState helper ([d20faac](https://github.com/rixlhq/terraform-provider-netcup/commit/d20faacf056a90f4cc1d870ebaf44210577ca6b4))
* **scp:** correct path parameter order and user_id substitution in generated data sources ([0e57f3c](https://github.com/rixlhq/terraform-provider-netcup/commit/0e57f3c5788df16eea1a5a3b119c674104440eaf))
* **scp:** generic CRUD path, body key mapping, and response root fallback ([ac9b1c4](https://github.com/rixlhq/terraform-provider-netcup/commit/ac9b1c438514781e0327efe5087b2d0a1b19867c))
* **scp:** guard nil queryBuilder in scp_server_action resource ([66cb843](https://github.com/rixlhq/terraform-provider-netcup/commit/66cb8436bff6b7d0fa1ff99ffd203d122eb8b983))
* **scp:** normalize camelCase JSON keys to snake_case and set correct POST content-type ([bc07f89](https://github.com/rixlhq/terraform-provider-netcup/commit/bc07f89c46f27dc72c54e38ebf27e5f5ce20bff2))
* **scp:** preserve known path parameters and config values in generic CRUD state ([8dd5a2e](https://github.com/rixlhq/terraform-provider-netcup/commit/8dd5a2e618c5afb24a833d2d689a97eb2e78682f))
* **scp:** route generic CRUD Read through readState to honor readFromList ([9b2c3d2](https://github.com/rixlhq/terraform-provider-netcup/commit/9b2c3d29a94b3a92ee61c87f3b2f105c1ee27eaa))
