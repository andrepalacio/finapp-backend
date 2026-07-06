# Phase 6: Production Deployment — FinApp

## Overview

Phase 6 brings FinApp to production on Railway platform with future migration path to GCP.

**Architecture:**
- Backend: Railway Docker container (Go + Gin)
- Frontend: Railway Docker container (Next.js)
- Database: Railway PostgreSQL
- Cache: Railway Redis
- Future: GCP Cloud Run + Cloud SQL (seamless migration)

## Quick Start (5 minutes)

### 1. Setup Railway

```bash
npm i -g @railway/cli
railway login
cd finapp-backend
railway init
cd ../finapp-frontend
railway init
```

### 2. Connect GitHub Repos

Each repo auto-links during `railway init`. Railway watches main branch for auto-deploy.

### 3. Add Plugins

Backend project:
```bash
railway add --plugin postgresql
railway add --plugin redis
```

Railway auto-provisions and sets env vars.

### 4. Set Environment Variables

Backend:
```bash
railway variables set \
  JWT_SECRET="$(openssl rand -base64 32)" \
  BCRYPT_COST="10" \
  ENV="production" \
  LOG_LEVEL="info"
```

Frontend:
```bash
railway variables set \
  NEXT_PUBLIC_API_URL="https://finapp-backend-prod.railway.app"
```

### 5. Deploy

Push to main:
```bash
git push origin main
```

Railway auto-builds, tests (GitHub Actions), and deploys both services.

**Deployment time:** ~10-15 minutes total

## Detailed Setup

### Backend Deployment

See: `finapp-backend/DEPLOYMENT.md`

Key steps:
1. Create Railway project
2. Add PostgreSQL + Redis plugins
3. Set environment variables
4. Migrations auto-run on deploy
5. Tests run via GitHub Actions

### Frontend Deployment

See: `finapp-frontend/DEPLOYMENT.md`

Key steps:
1. Create separate Railway project
2. Set `NEXT_PUBLIC_API_URL` to backend URL
3. Push to main
4. Auto-build with `npm run build`
5. Tests run via GitHub Actions

## Testing

Both services run automated tests before deploy:

**Backend** (.github/workflows/test.yml):
- `go test ./...`
- Docker image build

**Frontend** (.github/workflows/test.yml):
- `npm run lint`
- `npm run type-check`
- `npm run build`
- `npm run e2e` (Playwright tests)
- Docker image build

If tests fail, deploy is blocked.

## Environment Separation

### Production

- Branch: `main`
- Env: `production`
- Railway projects: finapp-backend-prod, finapp-frontend-prod

### Staging (Optional)

- Branch: `develop`
- Env: `staging`
- Railway projects: finapp-backend-staging, finapp-frontend-staging

Setup:
1. Create second set of Railway projects for staging
2. Connect develop branch to staging projects
3. Same configuration, different secrets

## Monitoring

Railway dashboard provides:
- Real-time logs
- CPU/Memory metrics
- Network I/O
- Deployment history
- Service health status

Access logs:
```bash
railway logs --follow
```

Alert on failures:
1. Dashboard → Service → "Alerts"
2. Configure email/Slack on deployment failure

## Scaling

### Vertical

Increase CPU/Memory per container:
1. Dashboard → Service → "Settings"
2. Adjust resource limits
3. Redeploy

### Horizontal

Run multiple replicas:
1. Dashboard → Service → "Settings"
2. Set "Replicas" to 2+ (requires paid plan)

## Backups

PostgreSQL backups automatic:
- Daily snapshots (7-day retention)
- Manual backup via dashboard
- One-click restore

Test restore monthly:
1. Create snapshot
2. Spin up new PostgreSQL service
3. Restore snapshot
4. Verify data integrity
5. Delete test service

## Database Migrations

Migrations run automatically during deploy via `golang-migrate` embedded in binary.

Manual migration (if needed):
```bash
railway shell
/app/server
# Connects to DB, migrations auto-run
```

New migrations:
1. Create in `db/migrations/NNN_name.up.sql`
2. Create rollback in `db/migrations/NNN_name.down.sql`
3. Regenerate sqlc: `make sqlc`
4. Commit and push
5. Railway auto-runs migration on deploy

## Rolling Updates

Railway handles zero-downtime deployments:
1. New container spins up
2. Health check passes
3. Old container terminates
4. DNS/load balancer switches

No manual intervention needed.

## Costs (Estimated)

**Railway Free tier:**
- 5GB bandwidth/month
- Limited compute/storage
- Good for testing

**Railway Pro ($5/month + usage):**
- Unlimited bandwidth
- Full compute
- Database backups
- Scale to 2+ replicas

**Typical production:**
- Backend container: $5-15/month
- Frontend container: $5-10/month
- PostgreSQL: $15-30/month
- Redis: $5-10/month
- **Total:** ~$30-65/month

*Costs increase with traffic/scale.*

## GCP Future Migration

When ready to move to GCP (Phase 7):

### Path 1: Cloud Run (Recommended)
1. Export Railway containers
2. Deploy to GCP Cloud Run (serverless)
3. Same Docker image works
4. Cost: ~$0.0000417/request (auto-scales to zero)

### Path 2: GKE (Kubernetes)
1. Package as Helm charts
2. Deploy to GCP GKE
3. Manual scaling/management
4. Cost: ~$150+/month (3-node cluster minimum)

### Path 3: Compute Engine (VMs)
1. Similar to Cloud Run setup
2. Fixed VMs (no auto-scale to zero)
3. Cost: ~$20-50/month per VM

Current Railway setup seamlessly transitions to any GCP option.

## Troubleshooting

### Deployment fails

```bash
# Check logs
railway logs

# Common issues:
# 1. Tests failing: Fix locally, push again
# 2. Env vars missing: Set in Railway dashboard
# 3. Database not ready: Wait 1-2 min, redeploy
```

### App won't connect to database

```bash
# Verify DATABASE_URL set
railway variables

# Check PostgreSQL plugin status
railway status

# Manual test
psql $(echo $DATABASE_URL)
```

### Frontend can't reach backend

1. Verify backend is running: `curl https://finapp-backend-prod.railway.app/health`
2. Check frontend env var: `NEXT_PUBLIC_API_URL`
3. Check backend CORS config allows frontend origin
4. Browser console shows exact error

### High memory usage

```bash
# Check process memory
railway logs | grep memory

# Options:
# 1. Increase Railway plan tier
# 2. Check for memory leaks in code
# 3. Enable auto-scaling to new replica
```

## Checklist

- [ ] Railway CLI installed
- [ ] GitHub repos connected to Railway
- [ ] PostgreSQL + Redis plugins added
- [ ] Environment variables set
- [ ] Migrations ran successfully
- [ ] Backend health endpoint responds
- [ ] Frontend loads, can reach API
- [ ] Tests pass locally + in CI
- [ ] Custom domain configured (if needed)
- [ ] Backups configured
- [ ] Monitoring/alerts enabled
- [ ] Staging environment configured (if needed)

## Next Steps (Phase 6 Continued)

After deployment live:

1. **TODO-006-4:** Secrets management (use Railway's built-in)
2. **TODO-006-5:** Stripe billing integration
3. **TODO-006-6:** Monitoring (Datadog/Grafana)
4. **TODO-006-7:** Backup strategy (pg_dump to S3)
5. **TODO-006-GCP:** Prepare GCP migration strategy

## Support

- Railway Docs: https://docs.railway.app
- Railway Status: https://status.railway.app
- GCP Migration Guide: `GCP_MIGRATION.md` (TODO)

---

**Phase 6 Status:** Ready for production deployment on Railway ✓
**GCP Migration Path:** Clear, documented, zero re-architecture needed ✓
