# Demo credentials

Default values (override in `ansible/vars/demo.yaml`). After install, use the cluster dashboard; Keycloak admin and Argo CD passwords are generated.

Developer and platform users authenticate through Keycloak realm **`backstage`**. Certificate identity for gitsign is the Keycloak **email** (`dev1@rhads.demo`), not a `@rhads.com` address.

| System | Username | Password |
| --- | --- | --- |
| Developer Hub / GitLab / Keycloak realm `backstage` | `dev1` `dev2` `dev3` | `backstage` |
| Developer Hub (RBAC admin) | `pe1` `pe2` `pe3` | `backstage` |
| GitLab | `root` | `backstage` |
| Quay | `quayadmin` | `backstage` |
| Nexus | `admin` | `admin123` |
| Vault UI | `vaultadmin` | `backstage` |
| TPA (realm `trustify`) | `tpa-admin` | `backstage` |
| Keycloak master | `temp-admin` | secret `keycloak/keycloak-initial-admin` |
| Argo CD | `admin` | secret `openshift-gitops/openshift-gitops-cluster` |
| ACS | `admin` | from secret `stackrox/central-htpasswd` key `password` (shown on the install dashboard) |
| Vault root token | — | secret `vault/vault-init` |
| Kafka Console | — | OpenShift login / console route `kafka-console.apps.<cluster>` (no extra user) |

Short platform URLs (after install): `https://rhdh.apps.<cluster>/`, `https://gitlab.apps.<cluster>/`, `https://quay.apps.<cluster>/`, `https://argocd.apps.<cluster>/`, `https://acs.apps.<cluster>/`, `https://nexus.apps.<cluster>/`, `https://vault.apps.<cluster>/`, `https://apicurio.apps.<cluster>/`, `https://kafka-console.apps.<cluster>/`, `https://dashboard.apps.<cluster>/`.

Apicurio Registry is anonymous for the demo (no extra user). Vault secrets for apps live under `secret/apps/{component_id}` and show on the Developer Hub Overview card (`vault.io/secrets-path`).

TPA API clients used by the pipeline and RHDA backend: Keycloak realm `trustify`, client `cli` (seeded at install). Do not treat demo passwords as production secrets.

**Note:** Trip planning uses the Red Hat `quarkus-langchain4j-bom` on RHBOQ 3.33. Kafka topics `flow-in` / `flow-out` carry the HITL CloudEvents.
