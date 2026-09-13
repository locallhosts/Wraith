# Deployment guide

## Manifest validation status

Before any of this: the manifests in `k8s/` were validated with
[`kubeconform`](https://github.com/yannh/kubeconform) v0.8.0 in strict mode
against real Kubernetes OpenAPI schemas —

```
Summary: 8 resources found in 4 files - Valid: 8, Invalid: 0, Errors: 0, Skipped: 0
```

That confirms they're schema-correct. It does **not** confirm they work
end-to-end against a live cluster (image needs to actually exist and be
pullable, PVC needs a `ReadWriteMany`-capable StorageClass, etc.) — that
part you'll do in Stage 2 below.

## Stage 1 — test locally before touching a real cluster

Don't apply these manifests to a shared/production cluster first. Spin up
a disposable local cluster, build the image locally, and prove the whole
thing works end to end. Two good options:

### Option A: kind (Kubernetes-in-Docker) — recommended

```bash
# install: https://kind.sigs.k8s.io/docs/user/quick-start/#installation
kind create cluster --name wraith

# build the backend image and load it directly into kind's node
# (skips needing a registry for local testing)
docker build -t wraith-backend:local ./backend-go
kind load docker-image wraith-backend:local --name wraith

# point the manifests at the locally-loaded image
sed -i 's#ghcr.io/yourname/wraith-backend:latest#wraith-backend:local#' k8s/02-backend.yaml
```

### Option B: minikube

```bash
minikube start
eval $(minikube docker-env)   # build directly into minikube's Docker daemon
docker build -t wraith-backend:local ./backend-go
sed -i 's#ghcr.io/yourname/wraith-backend:latest#wraith-backend:local#' k8s/02-backend.yaml
```

### Deploy to the local cluster

```bash
kubectl apply -f k8s/00-namespace-config.yaml

# fill in real values first — this is a template, not a real secret
cp k8s/01-secrets.template.yaml /tmp/wraith-secrets.yaml
# edit /tmp/wraith-secrets.yaml with real values, then:
kubectl apply -f /tmp/wraith-secrets.yaml
rm /tmp/wraith-secrets.yaml   # don't leave decrypted secrets on disk

kubectl apply -f k8s/02-backend.yaml
kubectl apply -f k8s/03-networkpolicy.yaml

# you'll also need Postgres, Elasticsearch, and Neo4j reachable from the
# cluster — for local testing, the fastest path is running those three
# via `docker compose up postgres elasticsearch neo4j` on your host and
# pointing the ConfigMap at your host's IP (kind/minikube can usually
# reach host.docker.internal or the host's LAN IP), OR apply minimal
# single-replica Deployments for them — not included here since production
# would use managed services (RDS, Elastic Cloud, Aura) instead of
# hand-rolled StatefulSets anyway.

kubectl -n wraith get pods -w
kubectl -n wraith logs -f deploy/wraith-backend
kubectl -n wraith port-forward svc/wraith-backend 8080:8080
curl http://localhost:8080/healthz
curl http://localhost:8080/readyz
```

If `/readyz` returns 503, it's almost always the Postgres connection —
check `WRAITH_DATABASE_URL` in the secret and confirm the backend pod can
resolve/reach that host from inside the cluster network.

### Tear down

```bash
kind delete cluster --name wraith
# or: minikube delete
```

## Stage 2 — Helm chart (same manifests, parameterized)

```bash
helm install wraith ./charts/wraith \
  --set image.repository=wraith-backend \
  --set image.tag=local \
  --namespace wraith --create-namespace

helm upgrade wraith ./charts/wraith --set image.tag=v0.2.0   # subsequent deploys
helm uninstall wraith --namespace wraith                      # teardown
```

Secrets are intentionally not templated in `values.yaml` — create the
`wraith-secrets` Secret the same way as Stage 1 before installing.

## Stage 3 — a real cluster (EKS/GKE/AKS/self-managed)

Once Stage 1 works end to end locally, the differences for a real cluster
are:

1. **Push the image to a real registry** (`ghcr.io`, ECR, GCR) instead of
   loading it into a local node — `docker build -t ghcr.io/you/wraith-backend:v0.1.0 . && docker push ...`,
   then reference that tag in `values.yaml`/`k8s/02-backend.yaml`.
2. **Real secrets management** — don't hand-apply a Secret manifest in a
   real environment; use an External Secrets Operator pointed at
   Vault/AWS Secrets Manager (see `docs/ENTERPRISE.md`).
3. **A `ReadWriteMany` StorageClass** for the `wraith-pipeline-output` PVC
   (EFS on EKS, Filestore on GKE, Azure Files on AKS) — `emptyDir` works
   for a single-replica local test but isn't durable and won't work with
   `ReadWriteMany` semantics across nodes.
4. **An Ingress + TLS** in front of `wraith-backend` — not included here
   since it's cloud/ingress-controller-specific (nginx-ingress + cert-manager,
   or your cloud's managed ingress).
5. **Managed Postgres/Elasticsearch/Neo4j** instead of anything
   self-hosted in-cluster, for the same reason you wouldn't hand-roll
   these for any production service.

## Which OS should you develop/test on before pushing to GitHub?

**Short answer: Linux, or Windows with WSL2. Avoid native Windows (cmd/PowerShell) for this project specifically.**

Why it matters here, concretely:

- **The Makefile, docker-compose healthchecks, and shell scripts in the CI
  workflows are bash.** They don't run natively in `cmd.exe` or
  PowerShell. WSL2 gives you a real bash environment with none of that
  friction.
- **GitHub Actions runs on `ubuntu-latest`.** Testing on Linux (or WSL2,
  which *is* a real Linux kernel, not an emulation layer) means what
  passes locally is what will pass in CI — no "works on my machine"
  surprises from path separators, line endings, or case-sensitivity
  differences (native Windows filesystems are case-insensitive; Linux
  isn't, and Go/Python imports occasionally break silently on Windows
  because of that).
- **Docker Desktop's WSL2 backend** gives you Linux containers with
  near-native performance; Docker Desktop's older Hyper-V backend (or
  native Windows containers) is slower and has had volume-mount
  permission quirks with Elasticsearch/Postgres data directories
  specifically.
- **Go and the ed25519/gofmt tooling work fine on native Windows too** —
  it's specifically the shell scripting and Docker layer where Windows
  causes friction for a project like this.

Concretely, if your daily driver is Windows:

```powershell
wsl --install                      # one-time, from an elevated PowerShell
# then do everything below inside the WSL2 Ubuntu shell, not PowerShell:
wsl
```

Then treat that WSL2 shell exactly like a Linux machine for everything in
this README — `docker compose up`, `go test`, `npm run build`, `git push`,
all of it. Docker Desktop on Windows auto-integrates with WSL2 if you
enable it in Docker Desktop's Settings → Resources → WSL Integration.

If you're on **macOS**, everything in this repo works natively — Docker
Desktop for Mac, Go, Node, and Python all behave like Linux for the
purposes of this project; no special steps needed beyond installing
Docker Desktop and Go/Node/Python via Homebrew.

One more thing worth doing regardless of OS: add a `.gitattributes` file
enforcing LF line endings, since Windows editors (including VS Code with
certain settings) default to CRLF, and CRLF in `.sh`/`Makefile`/YAML files
causes exactly the kind of "works locally, fails in CI" bug this whole
pipeline exists to prevent in detection rules — no reason to reintroduce
it in the tooling around them.
