# Migration proxy

The migration proxy mode provides a way to allow an existing registry instance to proxy requests for new repositories (not found in the existing registry storage backend) to a new registry instance backed by the upcoming metadata database. This is a required feature to enable the online migration of large registries.

## How it works

```mermaid
sequenceDiagram
  autonumber
  participant C as Client
  participant R1 as Proxy Registry (public)
  participant B1 as Proxy Registry<br>Storage Backend
  participant R2 as Target Registry (private)
  participant B2 as Target Registry<br>Storage Backend
	C->>R1: HTTP /v2/<name>/...
	R1->>B1: The repository<br>with <name> exists?
	B1->>R1: Yes/No
	alt Yes
    %%rect rgb(229, 255, 204)
      R1->>B1: Fulfil request
      R1->>C: Response
    %%end
	else No
		rect rgb(255, 255, 204)
      R1-xR2: Proxy request
      R2->>B2: Fulfil request
      R2->>R1: Response
      R1-xC: Proxy response
    end
end
```

A registry configured as a proxy validates every incoming request and, based on the repository path embedded in every request URL, determines if the target repository is known to the instance or not.

To determine if the repository is known, the proxy registry issues a stat request against its storage backend. In case it exists, the proxy instance fulfills the client request; otherwise, it proxies the request to another registry instance.

The target registry instance only receives requests for repositories that are unknown to the proxy registry, fulfills them, and returns the response through the proxy registry.

The target registry remains hidden from clients, the proxy registry remains the only public and visible entry point for the platform, and handles the request/response proxying internally, without needing an external reverse proxy in front of the two registries.

## Configuration

### Proxy instance(s)

The migration proxy mode should be enabled in the existing registry instance(s) only. Settings can be configured in the `migration.proxy` section. Please see the [configuration docs](https://gitlab.com/gitlab-org/container-registry/-/blob/master/docs/configuration.md#proxy) for a list of parameters and their possible values.

The [HTTP secret](https://gitlab.com/gitlab-org/container-registry/-/blob/master/docs/configuration.md#http) configuration parameter (`http.secret`) must be set to ensure that both proxy and target registries use the same piece of data to sign state. This should already be set for clustered environments.

#### Example

```yaml
http:
  secret: registrysecret
migration:
  proxy:
    enabled: true
    url: https://registry2.example.com
```

For development purposes an already ready [configuration file](https://gitlab.com/gitlab-org/container-registry/-/blob/master/config/proxy.yml) is present in the config folder.

### Target instance(s)

The migration proxy mode must _not_ be enabled in the new registry instance(s).

It's required to set the [HTTP secret](https://gitlab.com/gitlab-org/container-registry/-/blob/master/docs/configuration.md#http) configuration parameter (`http.secret`) to the same value as for the existing proxy registry, ensuring that both proxy and target registries share the same piece of data used to sign state.

It's also required to set the [HTTP host](https://gitlab.com/gitlab-org/container-registry/-/blob/master/docs/configuration.md#http) configuration parameter (`http.host`) to the fully-qualified URL of the old/proxy registry. This is used to generate `Location` headers delivered to clients in response to write requests. These `Location` headers must point to the public proxy registry and not to the private target registry.

#### Example

```yaml
http:
  host: https://registry.example.com
  secret: registrysecret
```

For development purposes an already ready [configuration file](https://gitlab.com/gitlab-org/container-registry/-/blob/master/config/proxied.yml) is present in the config folder
