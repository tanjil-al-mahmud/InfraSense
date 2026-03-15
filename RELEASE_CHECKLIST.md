# InfraSense Release Checklist

## Pre-Release Checks

### Testing
- [ ] All backend tests passing: `cd backend && go test ./...`
- [ ] All collector tests passing: `cd collectors/<name> && go test ./...` for each collector
- [ ] Frontend builds without errors: `cd frontend && npm run build`
- [ ] `.deb` package builds successfully: `make package-deb`
- [ ] `.rpm` package builds successfully: `make package-rpm`

### Installation Testing
- [ ] `install.sh` tested on Ubuntu 24.04 (fresh VM)
- [ ] `install.sh` tested on Rocky Linux 8 (fresh VM)
- [ ] Docker Compose stack starts cleanly: `cd deploy && docker compose -f docker-compose.dev.yml up -d`

### Service Health
- [ ] All services pass health checks:
  ```bash
  curl http://localhost:8080/health
  curl http://localhost:8428/health
  curl http://localhost:9090/-/healthy
  curl http://localhost:9093/-/healthy
  ```
- [ ] Default admin user created on fresh install
- [ ] API `/health` endpoint returns 200
- [ ] Metrics flowing to VictoriaMetrics:
  ```bash
  curl "http://localhost:8428/api/v1/query?query=up" | jq '.data.result | length'
  ```
- [ ] Grafana dashboards loading at `http://localhost:3000`

### Notifications
- [ ] Email notifications working (send test alert)
- [ ] Telegram notifications working (send test alert)
- [ ] Slack notifications working (send test alert)

### Release Metadata
- [ ] `CHANGELOG.md` updated with all changes for this version
- [ ] Version bumped in `packaging/deb/DEBIAN/control`
- [ ] Version bumped in `packaging/rpm/infrasense.spec`

---

## Release Steps

- [ ] Tag the release:
  ```bash
  git tag v1.0.0
  git push origin v1.0.0
  ```
- [ ] Verify GitHub Actions release workflow completes successfully
- [ ] Verify `infrasense_amd64.deb` is attached to the GitHub Release
- [ ] Verify `infrasense_x86_64.rpm` is attached to the GitHub Release
- [ ] Verify `install.sh` is attached to the GitHub Release
- [ ] Test install from GitHub Release URL:
  ```bash
  curl -fsSL https://github.com/infrasense/infrasense/releases/latest/download/install.sh | sudo bash
  ```

---

## Post-Release

- [ ] Update documentation site (https://docs.infrasense.io)
- [ ] Announce release (GitHub Discussions, community channels)
- [ ] Close milestone in GitHub Issues
- [ ] Create next milestone
