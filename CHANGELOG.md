# Changelog

## v1.0.21

- chore(deps): bump go to 1.26.5 (#99)
- refactor: register extension index via exthttp.RegisterRevisionedHandler (#100)

## v1.0.20

- Merge pull request #93 from steadybit/feat/add-claude-workflows
- chore(deps): bump github.com/jellydator/ttlcache/v3 from 3.4.0 to 3.4.1
- chore(deps): bump github.com/steadybit/action-kit/go/action_kit_sdk
- chore(deps): bump github.com/steadybit/discovery-kit/go/discovery_kit_sdk
- chore(deps): bump github.com/steadybit/event-kit/go/event_kit_api
- chore(deps): bump github.com/steadybit/extension-kit
- chore: silence SonarQube finding on secrets: inherit in Claude workflows
- fix: do not panic when delivering events while New Relic accounts could not be loaded, or when an incident has no description
- fix: escape values interpolated into New Relic GraphQL queries and build the request envelope with a JSON encoder, preventing query/JSON injection via muting-rule name/description, workload/entity guids and incident priorities
- fix: prevent GraphQL injection and handler panics in New Relic calls (#94)
- fix: set a timeout on the New Relic HTTP client so a slow or unresponsive API cannot block discovery, status checks or event delivery indefinitely

## v1.0.19

- chore(deps): bump alpine from 3.23 to 3.24
- chore(deps): bump golang.org/x/net to v0.55.0 (CVE-2026-39821) (#87)

## v1.0.18

- chore: update to go 1.26.4
- feat: add weekly auto patch-release workflow

## v1.0.17

- Support discovery group attribute via `STEADYBIT_EXTENSION_DISCOVERY_GROUP` env var (or `discovery.group` Helm value) — when set, the extension adds `steadybit.group=<value>` to every discovered target
- Update dependencies

## v1.0.16

- Bump Go to 1.26.3
- Update dependencies

## v1.0.15

- Bump Go to 1.25.9
- Support if-none-match for the extension list endpoint
- Update dependencies

## v1.0.14

- feat(chart): split image.name into image.registry + image.name
- Support global.priorityClassName
- Update alpine packages in Docker image to address CVEs
- Update dependencies

## v1.0.13

- Update dependencies

## v1.0.12

- Update dependencies

## v1.0.11

- Update dependencies

## v1.0.10

- Updated dependencies

## v1.0.9

- Updated dependencies
- Separate metrics for workload and incident check

## v1.0.8

- update dependencies
- Use uid instead of name for user statement in Dockerfile

## v1.0.7

- Set new `Technology` property in extension description
- Update dependencies (go 1.23)

## v1.0.6

- Update dependencies (go 1.22)

## v1.0.5

 - update dependencies

## v1.0.4

 - broken release

## v1.0.3

 - update dependencies

## v1.0.0

 - Initial release
