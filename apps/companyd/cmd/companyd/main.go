// Command companyd is the CompanyOS core service: Kernel, Application,
// Governance, Identity, Runtime, and Daemon co-located in one process,
// per ADR-0004 (docs/adr/ADR-0004-first-slice-technology-stack.md).
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Node-Features/company-os/apps/companyd/internal/adapters/httpapi"
	"github.com/Node-Features/company-os/apps/companyd/internal/adapters/identity/supabaseauth"
	"github.com/Node-Features/company-os/apps/companyd/internal/adapters/intelligence/anthropic"
	"github.com/Node-Features/company-os/apps/companyd/internal/adapters/intelligence/fallback"
	"github.com/Node-Features/company-os/apps/companyd/internal/adapters/intelligence/gemini"
	"github.com/Node-Features/company-os/apps/companyd/internal/adapters/intelligence/openai"
	"github.com/Node-Features/company-os/apps/companyd/internal/adapters/notify/realtime"
	"github.com/Node-Features/company-os/apps/companyd/internal/adapters/persistence/supabase"
	"github.com/Node-Features/company-os/apps/companyd/internal/application"
	"github.com/Node-Features/company-os/apps/companyd/internal/daemon"
	"github.com/Node-Features/company-os/apps/companyd/internal/fixtures"
	"github.com/Node-Features/company-os/apps/companyd/internal/identity"
	"github.com/Node-Features/company-os/apps/companyd/internal/runtime"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
)

// providerCooldown is how long a provider is skipped after a
// rate-limit/outage failure before internal/adapters/intelligence/fallback
// tries it again.
const providerCooldown = 60 * time.Second

// buildProviders constructs one fallback.Provider per configured API key,
// in priority order Gemini, OpenAI, Anthropic. Priority order is a
// deployment choice (first-slice plan), not an architectural ranking —
// docs/architecture/intelligence.md's evidence-based Router (M&E
// reliability, Finance cost, Governance eligibility) is Phase 3+ work; this
// is a much narrower "don't retry a provider that just rate-limited us"
// concern, scoped entirely inside the ProviderAdapter port.
func buildProviders(ctx context.Context) []fallback.Provider {
	var providers []fallback.Provider

	if key := os.Getenv("GEMINI_API_KEY"); key != "" {
		p, err := gemini.New(ctx, key)
		if err != nil {
			log.Printf("companyd: gemini adapter: %v", err)
		} else {
			providers = append(providers, fallback.Provider{Name: "gemini", Adapter: p})
		}
	}
	if key := os.Getenv("OPENAI_API_KEY"); key != "" {
		providers = append(providers, fallback.Provider{Name: "openai", Adapter: openai.New(key)})
	}
	if key := os.Getenv("ANTHROPIC_API_KEY"); key != "" {
		providers = append(providers, fallback.Provider{Name: "anthropic", Adapter: anthropic.New(key)})
	}

	if len(providers) == 0 {
		log.Println("companyd: no GEMINI_API_KEY/OPENAI_API_KEY/ANTHROPIC_API_KEY set — Runtime dispatch will fail every attempt")
	}
	return providers
}

func main() {
	// Boot stage 1: OS signal handling (ADR-0006). Cooperative shutdown is
	// wired before any dependency is touched.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Boot stage 2: config loading (ADR-0006; daemon.md "process-scoped
	// configuration loading").
	_ = godotenv.Load() // best-effort local dev convenience; production sets real env vars

	// Boot stage 3: logging bootstrap. stdlib log only today — no
	// structured logging, metrics, or tracing exists yet (ADR-0006 open
	// question).
	log.Println("companyd: starting")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Boot stage 4: adapter construction — persistence. Connects first;
	// everything below this point depends on it.
	// nil until DATABASE_URL is configured, so /health can honestly report
	// "not_configured" instead of a misleading connection failure.
	var db httpapi.DBPinger
	var pool *supabase.Pool
	if dsn := os.Getenv("DATABASE_URL"); dsn != "" {
		p, err := supabase.Connect(ctx, dsn)
		if err != nil {
			log.Printf("companyd: failed to connect to database: %v", err)
		} else {
			defer p.Close()
			db = p
			pool = p
		}
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", httpapi.HealthHandler(db))

	// Kernel, Application, Governance, Identity, Runtime, and Daemon are
	// co-located in this one process per ADR-0004. They require a database
	// connection; without one companyd still serves /health in a degraded
	// state rather than failing to start.
	//
	// Note on Kernel specifically (ADR-0006): there is no "start Kernel"
	// stage below. Kernel (internal/kernel/workflow) is stateless — per
	// docs/architecture/kernel.md it is "usable without a running
	// scheduler, worker, daemon, model, or external workflow engine." It is
	// invoked synchronously, per request, from inside Application (boot
	// stage 6 below), never constructed or started here.
	var d *daemon.Daemon
	if pool != nil {
		// Boot stage 4b: adapter construction — Identity. companyd never
		// trusts a client-asserted Principal (ROADMAP.md Phase 3 Slice 4):
		// every governed route below requires a verified Supabase JWT. Fails
		// soft like the database connection above — without SUPABASE_URL,
		// companyd logs it and leaves the governed routes unmounted rather
		// than starting in a state that can't actually authenticate anyone.
		var authn *supabaseauth.Verifier
		if supabaseURL := os.Getenv("SUPABASE_URL"); supabaseURL != "" {
			a, err := supabaseauth.New(ctx, supabaseURL)
			if err != nil {
				log.Printf("companyd: failed to construct Supabase Auth verifier: %v", err)
			} else {
				authn = a
			}
		} else {
			log.Println("companyd: no SUPABASE_URL — cannot verify Human authentication")
		}

		// Boot stage 4c: adapter construction — Identity, Principal
		// resolution (ROADMAP.md Phase 3 Slice 6). Loads the real, persisted
		// Organization instead of internal/fixtures.NewRegistry's Go
		// literal, failing loudly (not falling back to the literal) if the
		// migration's seed row is missing — same fail-closed shape as the
		// authn block above. principalResolver turns verified Human
		// evidence into a durable Principal on first sign-in; not consumed
		// by Application this slice, see fixtures.NewRegistryFromDB's doc.
		orgRepo := supabase.NewOrganizationRepository(pool)
		reg, err := fixtures.NewRegistryFromDB(ctx, orgRepo)
		if err != nil {
			log.Printf("companyd: failed to load Organization from database: %v — Application/Runtime not started", err)
		} else {
			principalResolver := identity.NewResolver(supabase.NewPrincipalRepository(pool), fixtures.OrganizationID)

			// Boot stage 5: adapter construction — intelligence providers.
			providers := buildProviders(ctx)
			provider := fallback.New(providers, providerCooldown)
			if len(providers) > 0 {
				log.Printf("companyd: intelligence providers (priority order, %s cooldown on rate-limit/outage): %s", providerCooldown, provider)
			}

			// Boot stage 6: Application construction. This is what invokes
			// Kernel decision functions later, per governed request — see the
			// note above.
			wakeup := make(chan uuid.UUID, 16)
			realtimePublisher := supabase.NewRealtimePublisher(pool)
			app := &application.Application{
				Repo:     supabase.NewWorkflowRepository(pool),
				Pending:  supabase.NewPendingCommandRepository(pool),
				Exec:     supabase.NewExecutionRepository(pool),
				Fixtures: reg,
				Notify:               wakeup,
				Research:             supabase.NewResearchRepository(pool),
				MonitoringEvaluation: supabase.NewMonitoringEvaluationRepository(pool),
				Finance:              supabase.NewFinanceRepository(pool),
				Objective:            supabase.NewObjectiveRepository(pool),
				Knowledge:            supabase.NewKnowledgeRepository(pool),
			}
			// Boot stage 7: Runtime construction — the "runtime subsystems"
			// stage (dispatch loop, leases, retries).
			rt := &runtime.Runtime{
				Exec:          supabase.NewExecutionRepository(pool),
				App:           app,
				Provider:      provider,
				ProviderName:  provider.String(),
				Fixtures:      reg,
				PollInterval:  5 * time.Second,
				LeaseDuration: 60 * time.Second,
				Wakeup:        wakeup,
				Notifier:      realtimePublisher,
			}
			// Boot stage 8: Daemon construction and start — supervises
			// Runtime's dispatch loop (daemon.md's process-lifecycle
			// ownership), not a handoff to Kernel.
			d = daemon.New(rt)

			// Boot stage 8b: realtime.Sweeper — Phase 1 Slice 8's push path.
			// Reads event_outbox (nothing did before this slice) and
			// broadcasts a minimal hint over Supabase Realtime (via
			// realtime.send() over the same DATABASE_URL pool, not a
			// separate credential — see supabase.RealtimePublisher) for
			// web's browser client to receive directly. Gated only on pool
			// != nil, the same condition everything else in this block
			// already depends on — no new env var required. Not supervised
			// by Daemon (which is scoped to Runtime's Scheduler contract) —
			// it's best-effort and idempotent-on-retry by construction, so
			// it's started and left to stop on ctx.Done() like the HTTP
			// server below, with no explicit drain needed.
			sweeper := &realtime.Sweeper{
				Outbox:       supabase.NewOutboxRepository(pool),
				Publisher:    realtimePublisher,
				OrgID:        fixtures.OrganizationID,
				PollInterval: time.Second,
				BatchSize:    50,
			}
			go sweeper.Start(ctx)
			log.Println("companyd: realtime sweeper started (event_outbox -> Supabase Realtime Broadcast)")

			// Boot stage 9: agent/workflow-layer entry points. Only
			// reachable once stages 4-8 above succeeded. Every route here
			// requires a verified Supabase JWT (Boot stage 4b) and resolves
			// it to a durable Principal (Boot stage 4c) — companyd never
			// trusts a client-asserted Principal.
			if authn != nil {
				withAuth := func(h http.HandlerFunc) http.HandlerFunc {
					return httpapi.RequireHumanAuth(authn, httpapi.ResolvePrincipal(principalResolver, h))
				}
				mux.HandleFunc("POST /v1/workflows", withAuth(httpapi.CreateWorkflowHandler(app)))
				mux.HandleFunc("POST /v1/workflows/{workflowId}/start", withAuth(httpapi.StartWorkflowHandler(app)))
				mux.HandleFunc("POST /v1/workflows/{workflowId}/results/accept", withAuth(httpapi.AcceptResultHandler(app)))
				mux.HandleFunc("POST /v1/workflows/{workflowId}/results/reject", withAuth(httpapi.RejectResultHandler(app)))
				mux.HandleFunc("POST /v1/workflows/{workflowId}/cancel", withAuth(httpapi.CancelWorkflowHandler(app)))
				mux.HandleFunc("GET /v1/workflows/{workflowId}", withAuth(httpapi.GetWorkflowHandler(app)))
				mux.HandleFunc("POST /v1/approvals/{approvalId}/decide", withAuth(httpapi.ResolveApprovalHandler(app)))

				// ROADMAP.md Phase 4 Slice 1 — docs/workflows/research-loop.md.
				mux.HandleFunc("POST /v1/research/signals", withAuth(httpapi.SubmitSignalHandler(app)))
				mux.HandleFunc("POST /v1/research/signals/{signalId}/questions", withAuth(httpapi.OpenResearchQuestionHandler(app)))
				mux.HandleFunc("POST /v1/research/questions/{questionId}/evidence", withAuth(httpapi.RecordEvidenceHandler(app)))
				mux.HandleFunc("POST /v1/research/questions/{questionId}/findings", withAuth(httpapi.PublishFindingHandler(app)))
				mux.HandleFunc("POST /v1/research/findings/{findingId}/recommendations", withAuth(httpapi.IssueRecommendationHandler(app)))
				mux.HandleFunc("GET /v1/research/questions/{questionId}", withAuth(httpapi.GetResearchQuestionHandler(app)))

				// ROADMAP.md Phase 4 Slice 2 — docs/departments/monitoring-evaluation.md.
				mux.HandleFunc("POST /v1/me/results/{resultId}/metrics", withAuth(httpapi.RecordMetricHandler(app)))
				mux.HandleFunc("POST /v1/me/evaluations", withAuth(httpapi.RunEvaluationHandler(app)))
				mux.HandleFunc("GET /v1/me/subjects/{subjectId}/performance-profile", withAuth(httpapi.GetPerformanceProfileHandler(app)))

				// ROADMAP.md Phase 4 Slice 3 — docs/departments/finance.md.
				mux.HandleFunc("POST /v1/finance/budgets", withAuth(httpapi.CreateBudgetHandler(app)))
				mux.HandleFunc("POST /v1/finance/constraints", withAuth(httpapi.CreateCostConstraintHandler(app)))
				mux.HandleFunc("POST /v1/finance/results/{resultId}/usage", withAuth(httpapi.RecordResourceUsageHandler(app)))
				mux.HandleFunc("POST /v1/finance/evaluations", withAuth(httpapi.RunResourceEvaluationHandler(app)))
				mux.HandleFunc("GET /v1/finance/constraint-status", withAuth(httpapi.GetConstraintStatusHandler(app)))
				mux.HandleFunc("GET /v1/finance/evaluations/{evaluationId}", withAuth(httpapi.GetResourceEvaluationHandler(app)))

				// ROADMAP.md Phase 4 Slice 4 — docs/architecture/departments.md's
				// Objective creation gate. Approval resolution reuses the
				// existing POST /v1/approvals/{approvalId}/decide route
				// above unchanged — ResolveApproval's resume dispatch is
				// already generic by CommandType.
				mux.HandleFunc("POST /v1/objectives/propose", withAuth(httpapi.ProposeObjectiveHandler(app)))
				mux.HandleFunc("GET /v1/objectives/{objectiveId}", withAuth(httpapi.GetObjectiveHandler(app)))

				// ROADMAP.md Phase 5 Slice 1 — docs/architecture/knowledge.md's
				// ingestion/versioning flow.
				mux.HandleFunc("POST /v1/knowledge/items/capture", withAuth(httpapi.CaptureKnowledgeCandidateHandler(app)))
				mux.HandleFunc("POST /v1/knowledge/items/{knowledgeItemId}/request-approval", withAuth(httpapi.RequestKnowledgeApprovalHandler(app)))
				mux.HandleFunc("GET /v1/knowledge/items/{knowledgeItemId}", withAuth(httpapi.GetKnowledgeItemHandler(app)))
			} else {
				log.Println("companyd: Supabase Auth verifier unavailable — /v1/workflows and /v1/approvals routes not mounted")
			}

			if err := d.Start(ctx); err != nil {
				log.Printf("companyd: daemon start error: %v", err)
			} else {
				log.Println("companyd: daemon started (Runtime dispatch loop running)")
			}
		}
	} else {
		log.Println("companyd: no DATABASE_URL — Kernel/Application/Runtime/Daemon not started, /v1/workflows routes unavailable")
	}

	// Boot stage 10: HTTP server starts listening.
	srv := &http.Server{Addr: ":" + port, Handler: mux}
	go func() {
		log.Printf("companyd: listening on :%s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("companyd: http server error: %v", err)
		}
	}()

	// Boot stage 11: block until shutdown signal, then drain and stop in
	// dependency-reverse order (daemon.md's supervision model).
	<-ctx.Done()
	log.Println("companyd: shutdown signal received, draining")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("companyd: http server shutdown error: %v", err)
	}

	if d != nil {
		if err := d.Shutdown(shutdownCtx); err != nil {
			log.Printf("companyd: daemon shutdown error: %v", err)
		}
	}
}