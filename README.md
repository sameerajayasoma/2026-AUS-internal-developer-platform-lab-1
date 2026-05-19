# 2026-AUS-internal-developer-platform-lab-1

A guided walkthrough of building an Internal Developer Platform on Kubernetes with [OpenChoreo](https://openchoreo.dev), a CNCF Sandbox project — from showcasing installation options, through defining component types, traits, and deployment workflows, to deploying services via a Backstage portal. You leave with a blueprint you can apply to your own cluster.

## Personas

Every action in the lab is performed as one of two users, so the boundary
between platform work and application work is clear at all times:

| Persona | Who they are | What they own |
|---|---|---|
| **Platform Engineer (PE)** | Owns the OpenChoreo deployment and the developer experience for the organization. | Environments, deployment pipelines, ComponentTypes, ClusterTraits, platform CRDs. |
| **Developer (Dev)** | Builds and ships application services. | Components, Workloads, releases against environments the PE has defined. |

The two parts of the lab map cleanly to these personas — Part 1 is performed
entirely as the PE, Part 2 entirely as the Dev.

## Agenda

The lab is split in two:

- **Part 1 — Platform Engineer role** : tour OpenChoreo's installation options, then configure a pre-deployed AWS environment to fit our example organization.
- **Part 2 — Developer self-service** : how application teams ship services onto the platform Part 1 sets up.

---

### Part 1 — Acting as the Platform Engineer

1. **Project intro on [openchoreo.dev](https://openchoreo.dev)**
   - Website, GitHub, ecosystem and Blog

2. **Installation options — showcase only**

   OpenChoreo documents three install paths. We tour them at a high level
   so attendees know what to pick at home; the actual lab work runs on a
   pre-deployed AWS environment, so **no installation is performed during
   the session**.

   - **[Quick Start](https://openchoreo.dev/docs/getting-started/quick-start-guide/)**
     — one command (`./install.sh --version v1.1.0`) inside a preconfigured
     devcontainer; spins up a k3d-in-Docker cluster with the **Control
     Plane** and **Data Plane**. Add `--with-build` for the Workflow Plane
     and `--with-observability` for the Observability Plane.
   - **[On K3D Locally](https://openchoreo.dev/docs/next/getting-started/try-it-out/on-k3d-locally/)**
     — step-by-step Helm install on a local k3d cluster. Installs all four
     planes (Control / Data / Workflow / Observability) in turn, with
     hands-on validation after each. This is the path our k3d walkthrough
     mirrors.
   - **[On Your Environment](https://openchoreo.dev/docs/next/getting-started/try-it-out/on-your-environment/)**
     — same step-by-step pattern on any Kubernetes target: **k3s, GKE,
     EKS, AKS, DOKS, or self-managed**. Requires `kubectl ≥ 1.32`, `helm
     ≥ 3.12`, LoadBalancer support, and a default StorageClass.

   The lab session itself runs against a pre-deployed AWS environment
   (the **On Your Environment** path, already executed for us).

3. **Tooling setup**
   - Install the **[`occ` CLI](https://openchoreo.dev/docs/getting-started/cli-installation/)** — OpenChoreo's command-line tool for managing platforms, projects, and components
   - Install the required Claude Code skills (PE-side: environments, traits, component types)
   - Add OpenChoreo's [two MCP servers](https://openchoreo.dev/docs/ai/mcp-servers/) to Claude Code:
     - **`openchoreo-cp`** — Control Plane MCP (deploy, configure, promote, troubleshoot)
     - **`openchoreo-obs`** — Observability MCP (logs, metrics, OpenTelemetry traces)

4. **Tour the pre-configured AWS environment**
   - Namespaces and Planes — see the [Architecture overview](https://openchoreo.dev/docs/overview/architecture)
   - What lives in each:
     - **Control Plane** — central orchestrator; reconciles platform and developer resources across the other planes
     - **Data Plane(s)** — isolated runtime and gateway topology where projects and components actually run
     - **Workflow Plane(s)** — executes CI workflows (build/test), GitOps reconciliation, and generic platform workflows
     - **Observability Plane(s)** — aggregates logs, metrics, and OpenTelemetry traces from the workflow and data planes
   - Walk through the CRDs — list them, then read through a sample CRD (e.g. a `Component` or `ComponentType`) to show how the platform API is shaped
   - Point out that the developer self-service experience (Part 2) is built entirely on top of this CRD API layer

5. **[Backstage portal](https://openchoreo.dev/docs/platform-engineer-guide/backstage-plugins/overview/) walkthrough**
   - Entity diagram filtered to the lab namespace
   - Environments (and the **DeploymentPipelines** that connect them)
   - ComponentTypes — OpenChoreo ships defaults for backend services, web applications, and scheduled tasks
   - Pick the **Web Application** ComponentType

6. **Configure the platform**

   The first two changes are authored as CRDs via the `openchoreo-cp` MCP server in Claude Code; the trait is then attached through the Backstage portal UI.

   - Add a new **`Environment`** (via Claude Code) — verify it and its **`DeploymentPipeline`** appear in the portal
   - Author a new **`ClusterTrait`** (via the Backstage portal) that injects security headers for web applications
   - Attach the ClusterTrait to the **Web Application** ComponentType **via the Backstage portal** — see [Component Types & Traits](https://openchoreo.dev/docs/platform-engineer-guide/component-types/overview/)


---

### Part 2 — Acting as the Developer

Covers the developer self-service flow on top of the platform Part 1 sets up.

## Demo app

The lab uses the URL shortener app in [`url-shortener-app/`](./url-shortener-app) (`frontend/` Go BFF + `backend/` Go API). It builds via Google Cloud Native Buildpacks — no Dockerfiles.
