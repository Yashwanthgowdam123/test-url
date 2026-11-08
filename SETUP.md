# MyURLs – Setup Guide

A lightweight URL dashboard. Go single binary in a `scratch` image (about 6 MB, near-zero CVEs), served over HTTPS by Caddy (automatic Let's Encrypt certificate). GitHub Actions builds, scans with Trivy, pushes to GHCR and deploys to your VM over SSH.

```
git push main ──▶ build image ──▶ Trivy gate ──▶ push to GHCR ──▶ SSH to VM ──▶ docker compose up
                                   (fail = stop)                    myurls.yashwanthgowdam.in
```

## Features
- Light and dark theme, live search, item counter.
- **User view:** open links only.
- **Admin view:** sign in with the admin password, then Add, Edit and Delete URLs.
- Data is stored in the Docker volume `myurls-data` and survives redeploys.

## Trivy gate
The pipeline fails and nothing is deployed unless all three hold:

| Rule | Limit |
|---|---|
| Total vulnerabilities | < 10 |
| Low + Medium | < 8 |
| High + Critical | < 2 |

## One-time prerequisites (done once, never again)

### 1. DNS
Create an **A record**: `myurls.yashwanthgowdam.in` → your VM public IP. Wait until `nslookup myurls.yashwanthgowdam.in` returns it.

### 2. VM
- Ubuntu or Debian with a public IP.
- Firewall / cloud security list open for **22, 80, 443** (on Oracle Cloud, also add ingress rules in the VCN).
- The SSH user must have **passwordless sudo** (default `ubuntu` user on most images). Check: `sudo -n true && echo ok`.
- Docker is installed automatically by the pipeline if missing.

### 3. GitHub repository
1. Create a new repo and push this project to the `main` branch.
2. Go to **Settings → Actions → General → Workflow permissions** and choose **Read and write permissions**.
3. Go to **Settings → Secrets and variables → Actions → New repository secret** and add:

| Secret | Value |
|---|---|
| `SSH_PRIVATE_KEY` | Entire content of your `.pem` file, including the BEGIN and END lines |
| `SSH_HOST` | VM public IP (or hostname) |
| `SSH_USER` | SSH username, for example `ubuntu` |
| `ADMIN_PASSWORD` | Password for the admin login (use a long one) |

## Deploy
```bash
git init && git add . && git commit -m "MyURLs"
git branch -M main
git remote add origin https://github.com/<you>/<repo>.git
git push -u origin main
```
That is the last manual step. Every later push to `main` redeploys automatically. Watch progress in the **Actions** tab.

First run takes a minute longer: Docker install, then Caddy requests the HTTPS certificate on first visit to the domain. Open **https://myurls.yashwanthgowdam.in/**.

## Using it
- Click the user menu (top right) → **Admin sign in** → enter `ADMIN_PASSWORD`.
- Use **Add URL**, the pencil to edit, the bin to delete.

## Troubleshooting
| Symptom | Fix |
|---|---|
| Pipeline fails at "Trivy gate" | Read the counts in the log. Rebuild after a base image update (re-run workflow) or raise the Go version in `Dockerfile`. |
| SSH step fails | Check the secret holds the full PEM, the user is right and port 22 is open. |
| `sudo: a password is required` | Give the user passwordless sudo: `echo "$USER ALL=(ALL) NOPASSWD:ALL" \| sudo tee /etc/sudoers.d/90-deploy` |
| HTTPS certificate not issued | DNS A record must point to the VM, and ports 80/443 must be open. Logs: `cd ~/myurls && sudo docker compose logs caddy` |
| App logs | `cd ~/myurls && sudo docker compose logs myurls` |
| Reset admin password | Change the `ADMIN_PASSWORD` secret and re-run the workflow. |

## Files
| File | Purpose |
|---|---|
| `main.go`, `web/index.html` | Backend API and UI |
| `Dockerfile` | Multi-stage build to `scratch`, non-root |
| `deploy/docker-compose.yml`, `deploy/Caddyfile` | App plus HTTPS reverse proxy for the domain |
| `.github/workflows/deploy.yml` | Build, Trivy gate, push, SSH deploy |

Notes: only the app image is gated by Trivy (the `caddy:2-alpine` proxy is the official upstream image). Data is saved in `/data/urls.json` inside the volume.
