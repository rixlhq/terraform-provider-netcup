# Changelog

## [2.1.0](https://github.com/rixlhq/terraform-provider-netcup/compare/v2.0.0...v2.1.0) (2026-10-18)


### Features

* enhance SCP error details, deprecate maintenance, validate snapshots, and add server backup time & state ([#43](https://github.com/rixlhq/terraform-provider-netcup/issues/43)) ([10802ad](https://github.com/rixlhq/terraform-provider-netcup/commit/10802ad98d9a006ef72aab08c00cdc0123b38b47))

## [2.0.0](https://github.com/rixlhq/terraform-provider-netcup/compare/v1.4.0...v2.0.0) (2026-10-18)


### ⚠ BREAKING CHANGES

* The deprecated legacy CCP DNS API (DomainWebservice JSON API) and its resources/data sources (`netcup_dns_record`, `netcup_dns_zone`, `netcup_dns_records`) have been completely removed. Provider configuration attributes `api_key`, `api_password`, `customer_number`, and `endpoint` are removed. DNS is now managed exclusively via the modern netcup CloudDNS REST API (`clouddns_api_key`). Added composite import support (`domain_id/name/type/rdata`) for `netcup_clouddns_record`.

#### How to Migrate from Legacy DNS to CloudDNS:
1. **Activate CloudDNS in CCP**: In the Netcup Customer Control Panel under Domains > DNS / CloudDNS, activate CloudDNS (existing records are preserved).
2. **Generate CloudDNS API Key**: Generate a 64-character bearer token under CCP > Master Data > API > API Keys, and configure `clouddns_api_key` (or set `NETCUP_CLOUDDNS_API_KEY`).
3. **Update Terraform Resources**: Replace `netcup_dns_record` with `netcup_clouddns_record`.
4. **Import Without Downtime**: Use composite syntax (`domain_id/name/type/rdata`):
   ```shell
   terraform state rm netcup_dns_record.example
   terraform import netcup_clouddns_record.example '12345/www/A/192.0.2.1'
   ```
   Or use declarative `import` blocks in Terraform >= 1.5. See the full [Migration Guide](docs/guides/migrating-to-clouddns.md) for details.

### Features

* remove legacy CCP DNS API and add CloudDNS migration support ([#39](https://github.com/rixlhq/terraform-provider-netcup/issues/39)) ([94fc0a1](https://github.com/rixlhq/terraform-provider-netcup/commit/94fc0a1ce6c1eff9c87b3d5b72c27f69ce3c7a3d))

## [1.4.0](https://github.com/rixlhq/terraform-provider-netcup/compare/v1.3.1...v1.4.0) (2026-10-18)


### Features

* **clouddns:** add dedicated DNS record support ([#34](https://github.com/rixlhq/terraform-provider-netcup/issues/34)) ([e0cc47d](https://github.com/rixlhq/terraform-provider-netcup/commit/e0cc47dfe9c13a6a5dde8371b1631c5f85b3fa7b))


### Bug Fixes

* **ci:** restore go-version-file in workflows and remove README note ([3b57544](https://github.com/rixlhq/terraform-provider-netcup/commit/3b5754409ad22d21214c5ac4adcf083e83f9ae3f))
* **dns:** report missing legacy CCP credentials ([#31](https://github.com/rixlhq/terraform-provider-netcup/issues/31)) ([2a60a8f](https://github.com/rixlhq/terraform-provider-netcup/commit/2a60a8f438d4416b9c69b7e18aa8a4b01628b32e))
* **scp:** align action paths and server state with the API ([cff5796](https://github.com/rixlhq/terraform-provider-netcup/commit/cff57962218fc6b9617140b3feb71d3e1c7a57f0))
* **scp:** align implementation with the API contract ([#32](https://github.com/rixlhq/terraform-provider-netcup/issues/32)) ([e432556](https://github.com/rixlhq/terraform-provider-netcup/commit/e4325567a3ace9ede394370b95f838d644552106))
* **scp:** await nested firewall update tasks ([e478a2a](https://github.com/rixlhq/terraform-provider-netcup/commit/e478a2a601b4a5977cbdb8c5a72970de86d79c2a))
* **scp:** escape path parameters and preserve snapshot import names ([de6a355](https://github.com/rixlhq/terraform-provider-netcup/commit/de6a35558e6592e3675aebefb795f371eeadbc55))
* **scp:** prevent duplicate actions, coordinate token refresh, and validate accepted tasks ([9edf332](https://github.com/rixlhq/terraform-provider-netcup/commit/9edf332fd57680738fec9ae1b013d9c0113cd1bf))
* **scp:** retain data source inputs and forward server sorting ([e9ca010](https://github.com/rixlhq/terraform-provider-netcup/commit/e9ca01096dad865bfd7e8c1d10a1cf52b0b70057))

## [1.3.1](https://github.com/rixlhq/terraform-provider-netcup/compare/v1.3.0...v1.3.1) (2026-10-18)


### Bug Fixes

* **firewall:** model rule addresses as sets ([#27](https://github.com/rixlhq/terraform-provider-netcup/issues/27)) ([0e89ffc](https://github.com/rixlhq/terraform-provider-netcup/commit/0e89ffcf8298cfb3815843b69bc19d18168c9174))
* **scp:** set Content-Type on bodyless requests ([#26](https://github.com/rixlhq/terraform-provider-netcup/issues/26)) ([def4788](https://github.com/rixlhq/terraform-provider-netcup/commit/def47885555b375f50228c3f1c2c1996d4aed21c))

## [1.3.0](https://github.com/rixlhq/terraform-provider-netcup/compare/v1.2.1...v1.3.0) (2026-10-10)


### Features

* **interface:** support import for scp_server_interface ([ca78a24](https://github.com/rixlhq/terraform-provider-netcup/commit/ca78a2457b8df595a44f3f48334c503df6aa0701))


### Bug Fixes

* **import:** reject empty MAC in server interface import IDs ([95bdb4b](https://github.com/rixlhq/terraform-provider-netcup/commit/95bdb4bb5b91fc433ad35eb604198cf21908ade2))
* **interface:** populate full state on refresh for import ([f6f3b8a](https://github.com/rixlhq/terraform-provider-netcup/commit/f6f3b8a4ff62d85d5b4cacc0b369f0aaa7bde6e6))
* **provider:** treat empty-string credentials as unconfigured ([06da876](https://github.com/rixlhq/terraform-provider-netcup/commit/06da87691cbe2f37f939b1d1e45a9a77f308a4d4))
* **review:** harden CI permissions and align example versions ([35a3bbf](https://github.com/rixlhq/terraform-provider-netcup/commit/35a3bbf4bd373b41c70227289ceb2d4c5d835e8c))

## [1.2.1](https://github.com/rixlhq/terraform-provider-netcup/compare/v1.2.0...v1.2.1) (2026-08-14)


### Bug Fixes

* **release:** run release only after CI succeeds, force tag creation, fix version.go annotation ([7f30c83](https://github.com/rixlhq/terraform-provider-netcup/commit/7f30c83846378679a5b75fd49314c6927e3e7181))

## [1.2.0](https://github.com/rixlhq/terraform-provider-netcup/compare/v1.1.1...v1.2.0) (2026-08-14)


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
* **release:** consolidate goreleaser into release-please.yml and remove duplicate release.yml ([127bc8d](https://github.com/rixlhq/terraform-provider-netcup/commit/127bc8de7a034acbe78525305ba27ae0dc6c6234))
* **release:** remove goreleaser-action default distribution and version ([4ddc9fd](https://github.com/rixlhq/terraform-provider-netcup/commit/4ddc9fde93b70cf4dd09380307e07eecf607bf5d))
* **release:** rename workflow file to release.yml ([085f472](https://github.com/rixlhq/terraform-provider-netcup/commit/085f4724500a97b6f281b58f3cf6e51e8468b6f2))
* **release:** run goreleaser as a downstream job in release-please workflow ([2216b4b](https://github.com/rixlhq/terraform-provider-netcup/commit/2216b4b467daaccbeb48ef2aa12a9447b8620bb3))
* **release:** trigger goreleaser on release:published and workflow_dispatch ([42a48ad](https://github.com/rixlhq/terraform-provider-netcup/commit/42a48ad47a27aebbf33e04cd7a81c58284efadaa))
* **release:** use goreleaser-action default version ([5cbe3de](https://github.com/rixlhq/terraform-provider-netcup/commit/5cbe3de310e7c339c21cc18e7f8ed9d484717dcc))
* **scp:** add createReadsBack for task-style create responses and clean up readState helper ([d20faac](https://github.com/rixlhq/terraform-provider-netcup/commit/d20faacf056a90f4cc1d870ebaf44210577ca6b4))
* **scp:** correct path parameter order and user_id substitution in generated data sources ([0e57f3c](https://github.com/rixlhq/terraform-provider-netcup/commit/0e57f3c5788df16eea1a5a3b119c674104440eaf))
* **scp:** generic CRUD path, body key mapping, and response root fallback ([ac9b1c4](https://github.com/rixlhq/terraform-provider-netcup/commit/ac9b1c438514781e0327efe5087b2d0a1b19867c))
* **scp:** guard nil queryBuilder in scp_server_action resource ([66cb843](https://github.com/rixlhq/terraform-provider-netcup/commit/66cb8436bff6b7d0fa1ff99ffd203d122eb8b983))
* **scp:** normalize camelCase JSON keys to snake_case and set correct POST content-type ([bc07f89](https://github.com/rixlhq/terraform-provider-netcup/commit/bc07f89c46f27dc72c54e38ebf27e5f5ce20bff2))
* **scp:** preserve known path parameters and config values in generic CRUD state ([8dd5a2e](https://github.com/rixlhq/terraform-provider-netcup/commit/8dd5a2e618c5afb24a833d2d689a97eb2e78682f))
* **scp:** route generic CRUD Read through readState to honor readFromList ([9b2c3d2](https://github.com/rixlhq/terraform-provider-netcup/commit/9b2c3d29a94b3a92ee61c87f3b2f105c1ee27eaa))

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
