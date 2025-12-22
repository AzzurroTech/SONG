SONG – Docker Demo Server
SONG (Simple Operations for Native Containers) is a tiny, zero‑dependency Go web application that lets you view, start, stop, and export Docker containers from a browser‑based UI.It is built, packaged, and maintained by Azzurro Technology Inc.

Features
- Live container listShows ID, image, command, creation time, status, ports, and container name.
- Stop a single containerRed “‑” button on each row (docker stop <id>).
- Stop all containersLarge floating red button (bottom‑right) stops every running container.
- Launch a new container Green “+” button opens a modal.
- Type any Docker‑Hub image, optionally browse the latest tags, and launch it (docker run -d <image>).
- Export as Compose
- One‑click download of a minimal docker-compose.yml that recreates the currently running containers (image, ports, environment).
- Zero external dependencies
- Pure Go standard library, compiled into a single static binary.Small runtime image
- Multi‑stage Dockerfile produces a ~15 MB distroless container.

All Docker interactions are performed via the Docker CLI (docker binary) invoked from Go (os/exec).
The server never stores credentials; it relies on the host’s Docker socket permissions.


Prerequisites
- Docker Engine20.10 or newer (host must expose the Docker socket to the container if you run SONG inside Docker).
- Go (only for building locally)1.22 or newer.
- User permissionsMust belong to the docker group or run the binary with sufficient privileges to talk to the Docker daemon.
- Network: Outbound HTTPS access to hub.docker.com for tag lookup (optional, only used in the “Add container” modal).

Getting Started
Run locally (binary)
# 1️⃣ Clone the repo (or copy the source files)
git clone https://github.com/azzurrotech/song-demo.git
cd song-demo

# 2️⃣ Build the binary
go build -o song .

# 3️⃣ Run (ensure your user can talk to Docker)
./song
You should see:
Server listening on http://localhost:8080

Open a browser at that address to interact with the UI.
Run inside Docker (recommended)

Why Docker?
Consistent environment – the same binary runs everywhere.
Small footprint – the final image is based on distroless/static, ~15 MB.

# Build the image (Dockerfile is included)
docker build -t song-demo:latest .

# Run the container, mounting the host Docker socket.
docker run -d \
  --name song-demo \
  -p 8080:8080 \
  -v /var/run/docker.sock:/var/run/docker.sock \
  song-demo:latest

Security note – mounting the Docker socket grants the container full control over the host Docker daemon. Use only on trusted hosts or in isolated environments.

Now visit http://localhost:8080 on the host.

Using the UI
UI ElementActionContainer tableLists all running containers.
Red “‑” button (per row)Stops that specific container (docker stop <ID>).
Large red floating “‑” button (bottom‑right)Stops all containers (docker stop $(docker ps -q)).
Green “+” button (bottom‑right)Opens a modal to launch a new container.
Image fieldType any Docker‑Hub image name (e.g., nginx).
Search tags buttonCalls Docker Hub API for up to 10 recent tags; click a tag to launch docker run -d <image>:<tag>.
Export (⤓) button (bottom‑right)Downloads a docker-compose.yml that recreates the displayed containers.
Page refresh Reload the browser to see the latest state (the server queries Docker on every request).

Exported docker-compose.yml
The exported file follows Compose v3 syntax and includes:

service name – deterministic (svc_<short‑container‑id>).
image – exact image reference from docker inspect.
ports – any host‑to‑container port mappings discovered via NetworkSettings.Ports.
environment – all environment variables defined for the container (Config.Env).

Only these three fields are emitted because they are universally supported and can be recreated safely.

Limitations – Volumes, custom networks, restart policies, and other advanced Docker options are not captured. You may manually edit the generated file to add them if needed.


Building the Image Yourself
The repository ships a multi‑stage Dockerfile:
# ------------------------------------------------------------
# Stage 1 – Builder
# ------------------------------------------------------------
FROM golang:1.23-alpine AS builder
RUN apk add --no-cache git
RUN addgroup -S appgroup && adduser -S appuser -G appgroup
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY *.go ./
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w" -o song .

# ------------------------------------------------------------
# Stage 2 – Runtime (distroless)
# ------------------------------------------------------------
FROM gcr.io/distroless/static:nonroot
COPY --from=builder /app/song /app/song
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/app/song"]
Stage 1 compiles a static binary named song.
Stage 2 copies that binary into a distroless image that runs as a non‑root user.
To rebuild:
docker build -t song-demo:custom .

Support & Services
Azzurro Technology Inc. offers professional services around SONG and any open‑source stack built with it:
- Custom Feature DevelopmentImplementation of additional Docker operations (volume management, network orchestration, authentication, etc.).
- Security review, RBAC integration, TLS termination, and sandboxed execution.
- Hands‑on sessions for DevOps teams on Docker automation, Compose generation, and container lifecycle management.
- Architecture reviews for container‑centric workloads, migration strategies, and cost optimisation.

Contact us
Website: https://azzurro.tech
Support email: info@azzurro.tech

Reach out to discuss your requirements, obtain a quote, or schedule a discovery call.

License
SONG is released under the MIT License. Feel free to use, modify, and redistribute it in commercial or open‑source projects. See the LICENSE file for full terms.

Enjoy a simple, fast way to manage Docker containers from the comfort of your browser—powered by SONG and backed by Azzurro Technology Inc.!
