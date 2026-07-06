# Deployment Guide — FinApp Backend (Railway + GCP)

## Architecture

- **Platform:** Railway (PaaS) for easy management, future migration to GCP
- **Backend:** Docker container (Golang + Gin)
- **Database:** Railway PostgreSQL plugin
- **Cache:** Railway Redis plugin
- **Environment:** Production

## Prerequisites

1. Railway account: https://railway.app
2. GitHub account with repo access
3. GCP account (for future migration)

## Step 1: Create Railway Project

```bash
# Install Railway CLI
npm i -g @railway/cli

# Login
railway login

# Link repo to Railway
railway init
```

Railway will create a `railway.json` file (already in repo).

## Step 2: Configure Environment Variables

Railway reads from `.env` or GitHub Actions secrets. Set these in Railway dashboard or via CLI:

```bash
railway variables set \
  DATABASE_URL="postgres://..." \
  REDIS_URL="redis://..." \
  JWT_SECRET="your-32-char-secret" \
  BCRYPT_COST="10" \
  ENV="production" \
  PORT="8080" \
  LOG_LEVEL="info"
```

Or in Railway dashboard:
1. Go to project settings
2. Add variables in "Environment" tab
3. Select "Production" environment

### Required Variables

| Variable | Value | Example |
|----------|-------|---------|
| DATABASE_URL | PostgreSQL connection string | postgres://user:pass@host:5432/finapp |
| REDIS_URL | Redis connection string | redis://host:6379/0 |
| JWT_SECRET | Minimum 32 characters | use: `openssl rand -base64 32` |
| BCRYPT_COST | Hash cost (10-12 recommended) | 10 |
| ENV | Environment name | production |
| PORT | App listen port | 8080 |
| LOG_LEVEL | Logging level | info |
| NEXT_PUBLIC_API_URL | Frontend API endpoint | https://api.finapp.com |

## Step 3: Add Database & Redis Plugins

Railway auto-provisions databases when you add plugins:

```bash
# Via Railway CLI
railway add --plugin postgresql
railway add --plugin redis
```

Or via dashboard:
1. Click "Add" in project
2. Select "PostgreSQL"
3. Select "Redis"

Railway auto-sets DATABASE_URL and REDIS_URL env vars.

## Step 4: Run Migrations

Migrations run automatically on first deploy via Dockerfile CMD:

```bash
# Or manually run in Railway shell:
railway shell
/app/server
# Once connected, schema auto-migrates (golang-migrate embedded)
```

## Step 5: Deploy

### Automatic (GitHub)

Railway auto-deploys on push to `main`:

1. Push code to main branch
2. Railway detects push
3. Runs GitHub Actions test workflow (test.yml)
4. If tests pass, Railway builds Docker image
5. Deploys to Railway infrastructure

```bash
git push origin main
```

### Manual (Railway CLI)

```bash
railway deploy
```

## Step 6: Verify Deployment

```bash
# Check logs
railway logs

# Test endpoint
curl https://api.finapp-prod.railway.app/health

# Check services
railway status
```

## Monitoring & Logs

Railway dashboard shows:
- Real-time logs (deployment, app output)
- CPU/Memory usage
- Network I/O
- Deployment history

View logs:
```bash
railway logs --follow
```

## Database Backups

Railway PostgreSQL includes:
- Daily automated backups (7-day retention by default)
- Manual backup option via dashboard
- One-click restore

To backup:
1. Dashboard → PostgreSQL plugin
2. "Backups" tab → "Create Backup"

## Scale Horizontally

Railway auto-scales based on CPU/memory. Configure limits:

1. Dashboard → Backend service → "Settings"
2. Set CPU limits, memory limits, replica count

## Environment Switching

For staging vs production:

1. Create separate Railway project for staging
2. Same configuration, different secrets
3. Deploy develop branch to staging project

## Troubleshooting

### Container fails to start
- Check logs: `railway logs`
- Verify env vars set correctly
- Ensure migrations ran: `SELECT * FROM schema_migrations;`

### Database connection refused
- Verify DATABASE_URL format
- Check PostgreSQL plugin is running
- Whitelist Railway IP if needed (should be automatic)

### High memory usage
- Check for memory leaks in code
- Increase Railway plan tier
- Scale up pod resources

## Future: Migrate to GCP

When ready to migrate to GCP:

1. Export Railway PostgreSQL dump
2. Set up GCP Cloud SQL PostgreSQL instance
3. Import dump
4. Update DATABASE_URL to GCP Cloud SQL connection string
5. Deploy GCP Cloud Run service (similar Docker setup)
6. Update DNS to point to GCP load balancer

This architecture makes GCP migration straightforward.

## Security Checklist

- [ ] JWT_SECRET is 32+ chars, random
- [ ] DATABASE_URL not in logs
- [ ] HTTPS enforced (Railway default)
- [ ] CORS configured for frontend domain
- [ ] Rate limiting enabled
- [ ] Database backups tested
- [ ] Secrets not in git (use .gitignore)

## Support

- Railway Docs: https://docs.railway.app
- Railway CLI: `railway help`
- GCP Migration Guide: See `GCP_MIGRATION.md` (future)
