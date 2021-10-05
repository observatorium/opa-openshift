# opa-openshift
An OPA-compatible API for making OpenShift access review requests.

## API

### POST /v1/data/{package}/{rule}

The `opa-openshift` HTTP server exposes a single endpoint of the [OPA Data API](https://www.openpolicyagent.org/docs/latest/rest-api/#data-api) and fulfills requests by translating them into [Kubernetes SubjectAccessReviews](https://docs.openshift.com/container-platform/latest/rest_api/authorization_apis/subjectaccessreview-authorization-k8s-io-v1.html). This endpoint expects an OPA [Input Document](https://www.openpolicyagent.org/docs/latest/kubernetes-primer/#input-document) in the body of the request with the following structure:

```json
{
    "input": {
        "groups": ["string"],
        "permission": "string",
        "resource": "string",
        "subject": "string",
        "tenant": "string"
    }
}
```

It returns a response with the following structure:

```json
{
    "result": "boolean"
}
```

Optionally, if a label set matcher is set by `--opa.matcher`, it returns a response with the following structure:

```json
{
    "result": {
        "allowed": "boolean",
        "data": [{
            "name": "string", 
            "type": "string", 
            "value": "string"
        }]
    }
}
```

The `data` section represents a PromQL-style Label matcher:

| Key   | Description                            |
| ---   | :--                                    |
| name  | The value of the `--opa.matcher` flag  |
| type  | The value is per default `MatchRegexp` |
| value | A comma-separated list of OpenShift projects the subject has access to. |

### Design

The `opa-openshift` authorization process translates in general an [OPA Data Request V1](https://www.openpolicyagent.org/docs/latest/rest-api/#data-api) into a
[SubjectAccessReview](https://docs.openshift.com/container-platform/latest/rest_api/authorization_apis/subjectaccessreview-authorization-k8s-io-v1.html). 

To authorize the subject against a resource for a tenant, the server requires a list of tenant to api group mappings, e.g.:

```shell
./opa-openshift \
   --openshift-mappings=tenant-a=observatorium.openshift.io
```

Assuming an input data request, e.g.:
```json
{
    "input": {
        "groups": ["system:authenticated"],
        "permission": "read",
        "resource": "resource-name",
        "subject": "k8suser",
        "tenant": "tenant-a"
    }
}
```

The input is translated to the following `SelfSubjectAccessReview`:

```json
{
    "apiVersion": "authorization.k8s.io/v1",
    "kind": "SelfSubjectAccessReview",
    "spec": {
        "resourceAttributes": {
            "resource": "tenant-a",
            "resourceName": "resource-name",
            "verb": "get",
            "apiGroup": "observatorium.openshift.io"
        }
    }
}
```

If the subject's `k8suser` role bindings allow an access to the resource description in `resourceAttributes`, then the `SelfSubjectAccessReview`'s  status will result in:

```json
{
    "apiVersion": "authorization.k8s.io/v1",
    "kind": "SelfSubjectAccessReview",
    "spec": {},
    "status": {                                                                                                                                                                                  
        "allowed": true,
        "reason": "RBAC: allowed by RoleBinding \"tenant-a-binding\" of Role \"tenant-a\" to User \"k8suser\""
    }
}
```

## Usage

[embedmd]:# (tmp/help.txt)
```txt
Usage of ./opa-openshift:
      --debug.name string                      A name to add as a prefix to log lines. (default "opa-openshift")
      --log.format string                      The log format to use. Options: 'logfmt', 'json'. (default "logfmt")
      --log.level string                       The log filtering level. Options: 'error', 'warn', 'info', 'debug'. (default "info")
      --memcached strings                      One or more Memcached server addresses.
      --memcached.expire int32                 Time after which keys stored in Memcached should expire, given in seconds. (default 3600)
      --memcached.interval int32               The interval at which to update the Memcached DNS, given in seconds; use 0 to disable. (default 10)
      --opa.matcher string                     The label key of the OPA label matcher returned to the requesting client.
      --opa.package string                     The name of the OPA package that opa-openshift should implement, see https://www.openpolicyagent.org/docs/latest/policy-language/#packages.
      --opa.rule string                        The name of the OPA rule for which opa-openshift should provide a result, see https://www.openpolicyagent.org/docs/latest/policy-language/#rules. (default "allow")
      --openshift.kubeconfig string            A path to the kubeconfig against to use for authorizing client requests.
      --openshift.mappings strings             A map of tenantIDs to resource api groups to check to apply a given role to a user, e.g. tenant-a=observatorium.openshift.io
      --tls.cipher-suites string               Comma-separated list of cipher suites for the server. Values are from tls package constants (https://golang.org/pkg/crypto/tls/#pkg-constants). If omitted, the default Go cipher suites will be used. Note that TLS 1.3 ciphersuites are not configurable.
      --tls.internal.server.ca-file string     File containing the TLS CA against which to verify servers. If no server CA is specified, the client will use the system certificates.
      --tls.internal.server.cert-file string   File containing the default x509 Certificate for internal HTTPS. Leave blank to disable TLS.
      --tls.internal.server.key-file string    File containing the default x509 private key matching --tls.internal.server.cert-file. Leave blank to disable TLS.
      --tls.min-version string                 Minimum TLS version supported. Value must match version names from https://golang.org/pkg/crypto/tls/#pkg-constants. (default "VersionTLS13")
      --tls.server.cert-file string            File containing the default x509 Certificate for HTTPS. Leave blank to disable TLS.
      --tls.server.key-file string             File containing the default x509 private key matching --tls.server.cert-file. Leave blank to disable TLS.
      --web.healthchecks.url string            The URL against which to run healthchecks. (default "http://localhost:8080")
      --web.internal.listen string             The address on which the internal server listens. (default ":8081")
      --web.listen string                      The address on which the public server listens. (default ":8080")
```
