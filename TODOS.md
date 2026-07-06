# FinApp — TODOs

Items captured during engineering reviews. Each entry has enough context to pick up cold.

---

## Auth & Users

### TODO-001: Password change endpoint ✓ DONE
**What:** `PUT /api/v1/user/password` — allow users to change their own password.
**Status:** ✓ COMPLETED 2026-07-05
**Implemented:**
- Handler validates current password with bcrypt.CompareHashAndPassword
- Hashes new password, updates in DB via UpdateUserPassword query
- Returns 204 No Content on success
- 3 unit tests cover success, wrong password, validation cases
**Commit:** `e5e9a16` in finapp-backend
**Tests:** All passing (user_service + handlers)

---

### TODO-002: Account deletion endpoint ✓ DONE
**What:** `DELETE /api/v1/user` — hard delete of the authenticated user's account.
**Status:** ✓ COMPLETED 2026-07-05
**Implemented:**
- Hard delete removes user from database (cascades via foreign keys)
- Service method Delete(ctx, userID) calls repository
- Handler returns 204 No Content on success
- Unit tests verify delete flow
**Commit:** `e5e9a16` in finapp-backend
**Tests:** All passing (user_service + handlers)
**Note:** Hard delete chosen — simpler for Phase 6, no soft delete complexity

---

## Fase 6 — Producción & Base SaaS

### TODO-006-1: Pipeline CI/CD con GitHub Actions ✓ DONE
**What:** Workflow `.github/workflows/test.yml` — Test → Build Docker image → Auto-deploy via Railway.
**Status:** ✓ COMPLETED 2026-07-05
**Implemented:**
- Backend: go test ./... + docker build on main/develop push
- Frontend: npm lint, type-check, build, e2e (Playwright) + docker build
- Auto-triggers on push (no manual deploy step)
- Railway handles automatic deployment if tests pass
**Commit:** `23d3c12` (backend), `c771383` (frontend)
**Env:** PostgreSQL + Redis services spin up in CI for testing

---

### TODO-006-2: Servidor de producción ✓ DONE
**What:** Configure production platform.
**Status:** ✓ COMPLETED 2026-07-05 — RAILWAY SELECTED
**Decision:** Railway.app (PaaS) with GCP migration path
**Implementation:**
- railway.json config in backend + frontend
- Procfile for process definition
- PostgreSQL + Redis plugins provisioned via Railway
- Zero-downtime deployments
- Auto-scales based on traffic
- Cost: ~$30-65/month (startup tier)
**GCP Future Path:** Export containers → Cloud Run (serverless) or GKE (managed Kubernetes)
**Docs:** PHASE_6_DEPLOYMENT.md in each repo
**Commit:** `23d3c12` (backend), `c771383` (frontend)

---

### TODO-006-3: Nginx + SSL ✓ N/A
**What:** Reverse proxy + HTTPS.
**Status:** ✓ NOT NEEDED — Railway handles automatically
**Why:** Railway provides automatic SSL/TLS termination, load balancing, and DNS management.
- No manual Nginx config required
- Free HTTPS certificates (managed)
- Auto-renewal included
- Custom domains supported via Railway dashboard

---

### TODO-006-4: Secrets en producción ⏳ PENDING
**What:** Gestión segura de variables de entorno en prod.
**Status:** ⏳ PENDING (Ready to implement)
**Current:** Railway dashboard secrets + GitHub Actions secrets
**Scope:**
- Use Railway env var management (built-in, no external service)
- Variables: `DATABASE_URL` (auto-set), `REDIS_URL` (auto-set), `JWT_SECRET`, `NEXT_PUBLIC_API_URL`
- Never commit `.env` prod (already in .gitignore)
- Document Railway secrets procedure
**Depends on:** TODO-006-2 ✓ DONE
**Next step:** Document Railway secrets best practices

---

### TODO-006-5: Billing básico con Stripe ⏳ PENDING
**What:** Planes Free y Pro con límites definidos.
**Status:** ⏳ PENDING (Scope TBD)
**Scope:**
- Definir límites: Free = 1 workspace, 200 tx/mes; Pro = ilimitado
- Stripe Checkout integration + webhooks
- Columna `plan` en users/workspaces table
- Middleware para validar límites antes de crear recursos
- Stripe webhook handler para plan changes
**Depends on:** TODO-002 ✓ DONE + TODO-006-2 ✓ DONE
**Note:** Implementar después de TODO-006-4 (no bloquea deploy)
**Effort:** ~2-3 days (schema + Stripe integration + tests)

---

### TODO-006-6: Monitoreo — logs y alertas ⏳ PENDING
**What:** Observabilidad básica en producción.
**Status:** ⏳ PENDING (Scope TBD)
**Options:** 
- Datadog free tier (simplest start)
- Grafana + Loki self-hosted (more control)
- Railway built-in logs + alerts (minimal setup)
**Scope:**
- Structured logging backend (JSON, already using zap)
- Alerts: 5xx error rate > 1%, p99 latency > 5s, disk usage > 80%
- Email/Slack notifications
- Dashboard for key metrics
**Depends on:** TODO-006-2 ✓ DONE
**Recommendation:** Start with Datadog free tier or Railway alerts

---

### TODO-006-7: Backup automatizado de PostgreSQL ⏳ PENDING
**What:** Automated daily backups a S3 o compatible.
**Status:** ⏳ PENDING (Ready to implement)
**Current:** Railway PostgreSQL auto-backups (7-day retention, included)
**Scope:**
- Railway backups: automatic, daily snapshots (included)
- Optional: export to S3/R2/B2 for long-term retention (30+ days)
- Document restore procedure + test monthly
- Railway backup UI: dashboard → PostgreSQL → Backups
**Depends on:** TODO-006-2 ✓ DONE
**Next step:** Test Railway backup + restore flow, consider S3 export if >7 days needed
