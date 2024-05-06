<img src="./logo.png" height="80" align="right" alt="New Relic logo">

# Steadybit extension-newrelic

A [Steadybit](https://www.steadybit.com/) extension for [New Relic](https://newrelic.com).

Learn about the capabilities of this extension in our [Reliability Hub](https://hub.steadybit.com/extension/com.steadybit.extension_newrelic).

## Configuration

| Environment Variable                                  | Helm value                             | Meaning                                                                                                                            | Required | Default |
|-------------------------------------------------------|----------------------------------------|------------------------------------------------------------------------------------------------------------------------------------|----------|---------|
| `STEADYBIT_EXTENSION_API_BASE_URL`                    | `newrelic.apiBaseUrl`                  | The New Relic API Base Url, like 'https://api.newrelic.com' or 'https://api.eu.newrelic.com'                                       | yes      |         |
| `STEADYBIT_EXTENSION_API_KEY`                         | `newrelic.apiKey`                      | The New Relic [API Key](https://docs.newrelic.com/docs/apis/intro-apis/new-relic-api-keys/), Type: USER                            | yes      |         |
| `STEADYBIT_EXTENSION_INSIGHTS_COLLECTOR_API_BASE_URL` | `newrelic.insightsCollectorApiBaseUrl` | The New Relic Ingest API Base Url, like 'https://insights-collector.newrelic.com' or 'https://insights-collector.eu01.nr-data.net' | yes      |         |
| `STEADYBIT_EXTENSION_INSIGHTS_COLLECTOR_API_KEY`      | `newrelic.insightsCollectorApiKey`     | The New Relic [Ingest API Key](https://docs.newrelic.com/docs/apis/intro-apis/new-relic-api-keys/), Type: INGEST - LICENSE         | yes      |         |

The extension supports all environment variables provided by [steadybit/extension-kit](https://github.com/steadybit/extension-kit#environment-variables).

When installed as linux package this configuration is in`/etc/steadybit/extension-newrelic`.

## Installation

### Using Docker

```sh
docker run \
  --rm \
  -p 8090 \
  --name steadybit-extension-newrelic \
  ghcr.io/steadybit/extension-newrelic:latest
```

### Using Helm in Kubernetes

```sh
helm repo add steadybit-extension-newrelic https://steadybit.github.io/extension-newrelic
helm repo update
helm upgrade steadybit-extension-newrelic \
    --install \
    --wait \
    --timeout 5m0s \
    --create-namespace \
    --namespace steadybit-agent \
    steadybit-extension-newrelic/steadybit-extension-newrelic
```

## Register the extension

Make sure to register the extension at the steadybit platform. Please refer to
the [documentation](https://docs.steadybit.com/integrate-with-steadybit/extensions/extension-installation) for more information.
