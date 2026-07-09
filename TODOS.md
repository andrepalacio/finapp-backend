# FinApp — TODOs

Items captured during engineering reviews. Each entry has enough context to pick up cold.

---

## Fase 6 — Producción & Base SaaS

### TODO-006-5: Billing básico con Stripe ⏳ PENDING
**What:** Planes Free y Pro con límites definidos.
**Status:** ⏳ PENDING (Scope TBD)
**Scope:**
- Definir límites: Free = 1 workspace, 200 tx/mes; Pro = ilimitado
- Stripe Checkout integration + webhooks
- Columna `plan` en users/workspaces table
- Middleware para validar límites antes de crear recursos
- Stripe webhook handler para plan changes
**Depends on:** account deletion endpoint (ya implementado) + deploy en Railway (ya en producción)
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
**Depends on:** deploy en Railway (ya en producción)
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
**Depends on:** deploy en Railway (ya en producción)
**Next step:** Test Railway backup + restore flow, consider S3 export if >7 days needed

---

## Testing (2026-07-09 review)

### TODO-007: Integration tests para repos CRUD-puro restantes ⏳ PENDING
**What:** `category_repository`, `budget_repository`, `invitation_repository`, `user_repository` sin integration test.
**Status:** ⏳ PENDING (baja prioridad — CRUD puro sqlc, ya cubierto vía mocks en internal/services)
**Current:** `workspace`/`debt`/`savings`/`transaction` repos ya tienen integration test real contra Postgres (`make test-integration`, harness en `internal/repositories/testdb_integration_test.go`).
**Scope:** Mismo patrón — `setupTestDB(t)`, seed vía `createTestUser`/`createTestWorkspace`, CRUD + not-found paths.
**Next step:** Extender cuando estos repos ganen lógica no trivial (hoy son passthrough directo a sqlc).
