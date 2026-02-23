# Heimdall — Leadership Demo Script

**Total time:** 15 minutes
**URL:** http://localhost:3000
**Audience:** Engineering leadership, CPO, CEO
**Goal:** Make them feel the problem, then show we've solved it

---

## Before you start

Open three browser tabs in order:
1. http://localhost:3000 (Control Center)
2. http://localhost:3000/catalog (Source Catalog)
3. http://localhost:3000/changes (Change Detection)

Make sure the worker is running — new change events fire every 30s, which creates the live moment.

---

## Opening (60 seconds) — Make them feel the problem

> "I want to start with a number. The average enterprise has **60+ source systems**. Salesforce, Stripe, Postgres, HubSpot, Marketo, Zendesk — you know them. How many do most companies actually have connected to their data platform?"
>
> *[pause]*
>
> "Fifteen. One in four. Not because they don't want the data. Because every new connector feels like a risk they can't govern. Their CISO asks: 'What PII will you extract?' And the honest answer today is: 'We won't know until we extract it.' Request denied.
>
> That paradox — you need to extract the data to know if you should extract the data — is what Heimdall eliminates."

---

## Act 1 — Day Zero (3 minutes)

### The setup

> "Let me show you what happens when a company connects Heimdall for the first time."

**→ Open Control Center** (http://localhost:3000)

Point to the stats tiles:

> "Seven source systems are connected here — three with full data sync, four in **metadata-only mode**. That means Fivetran crawled their schemas, tables, and columns without extracting a single row. Free. Always on."

Point to the source health panel on the right:

> "You can see Salesforce, Stripe, and Postgres are actively syncing. HubSpot, Marketo, Zendesk, Intercom — connected in metadata-only mode. We can see everything across all of them. Zero data moved for those four."

---

**→ Switch to Source Catalog** (http://localhost:3000/catalog)

Click **HubSpot Marketing** in the left sidebar:

> "Here's HubSpot. Connected this afternoon. No data has moved. But look — we can already see every schema, every table, every column across their entire instance."

Click **hubspot.contacts** in the table list:

> "contacts table. Seven columns. And because we haven't run PII classification yet, these are unclassified — but watch what happens when we do."

Click **🔍 Run PII Scan** *(top right)*:

> "We just asked Claude to classify every column in this source for PII. It's running in the background right now — no data extraction needed, just the column names, types, and statistics we already have from the metadata crawl."

*[While it runs, switch to Salesforce]*

Click **Salesforce CRM** in the sidebar, then **salesforce.contact**:

> "Salesforce — which is actively syncing. Look at this table. `email`, `phone`, `mailing_street`, `birthdate` — if classification has run on this source, you'll see these flagged automatically."

> "This is what our CISO got blocked on for six months. 'What PII will be extracted?' With Heimdall, that question is answered **before you extract anything**."

---

## Act 2 — The Breaking Change (4 minutes)

### Build the tension first

> "Now let me show you the moment every data team dreads. A Salesforce admin — let's call him Kenji — renames a picklist value from 'Enterprise' to 'Enterprise Tier'. Completely reasonable change from his perspective. He doesn't know three dbt models depend on it."

> "Without Heimdall: the change happens Monday morning. Fivetran syncs it at 11am. dbt builds at 2pm — no SQL error, because it's a value change, not a schema error. Tuesday morning, the Sales VP opens her revenue dashboard. **Enterprise segment shows zero dollars.** By the time Marcus traces it back to Kenji's change, it's Tuesday 3pm. Twenty-four hours of bad data."

**→ Switch to Change Detection** (http://localhost:3000/changes)

> "With Heimdall — same Monday morning, Kenji renames the field."

*[Point to the live indicator at the top — `⬤ live`]*

> "Our crawler picks it up within two minutes. Watch the feed."

*[Wait for a new event to appear — they fire every 30s. Or point to the existing CRITICAL events already visible.]*

Click the **CRITICAL** event for `stage_name` / `salesforce.opportunity`:

> "There it is. Monday 10:02am. Before anyone noticed."

Point to the right panel — blast radius:

> "Heimdall shows us the exact change, Claude's severity classification, and the **blast radius** — every downstream dbt model, every MetricFlow metric, every dashboard that could break."

Point to the affected models badges: `fct_revenue`, `dim_accounts`, `fct_pipeline`

Point to the affected metrics: `mrr_metric`, `pipeline_by_segment`

Point to the affected dashboard: `Sales VP Dashboard`

> "Three models. Two metrics. One dashboard. And we caught it before a single build ran."

Point to the AI analysis panel:

> "Claude explains exactly why this is critical — the fct_revenue model filters on stage_name, so it would show zero revenue for the Enterprise segment until fixed. And here's the remediation: add a coalesce to handle both values, then update the source contract."

Point to the red **'Extraction Blocked'** badge:

> "Heimdall blocked the next extraction automatically. Data won't move until the engineer marks this resolved."

Click **✓ Mark as Resolved & Unblock**:

> "Marcus fixes it, marks resolved. The Sales VP's dashboard just works. She never knew."

*[pause for effect]*

> "Monday 10:15am. Fixed. Fifteen minutes. Nobody noticed."

---

## Act 3 — Governance Unlock (3 minutes)

> "The other story I want to show you is the CISO conversation."

**→ Switch to Source Catalog** (http://localhost:3000/catalog)

Click **Salesforce CRM**, then **salesforce.contact**:

> "Let's say this is a healthcare system — we've connected it in metadata-only mode. Before a single byte of data moves, the CISO asks: 'What PII will be extracted?'"

*[If PII scan has completed, columns will show EMAIL, PHONE, ADDRESS, DOB badges. If not, use Salesforce which is full-sync and may have classifications.]*

> "Heimdall answers that question right here. `email` — classified as EMAIL with 96% confidence. `phone` — PHONE. `mailing_street` — ADDRESS. `birthdate` — DOB. All classified by AI, from metadata alone."

**→ Switch to Policy Engine** (http://localhost:3000/policies)

Point to the YAML editor:

> "This is the policy our data governance lead defined. SSNs are completely blocked. Emails and phones require approval. Names and addresses are extracted with masking. **Defined once. Applies to all current and future connections. Versioned in git.**"

Click **▶ Evaluate Policy** after selecting `salesforce.contact` from the dropdowns:

> "Let's see what happens if we try to extract the contacts table right now, with this policy applied."

*[Decision badge appears — DENY or PARTIAL depending on what's classified]*

> "The decision engine evaluates every column against the policy and returns an instant verdict — with exactly which columns are blocked and why. This is the audit trail the CISO can take to their board."

---

## Act 4 — Compliance Audit (2 minutes)

**→ Switch to Compliance Audit** (http://localhost:3000/audit)

> "Last thing. SOC2 auditor walks in. 'Show me what personal data you extract, from where, who approved it.'"

> "Without Heimdall: seven days. Email Priya for connector list, cross-reference spreadsheets, query Snowflake for column names, compile a document, discover you missed a connector."

Click the **PII Report** tab:

> "With Heimdall: fifteen minutes."

*[PII report loads — shows summary tiles and table of every classified column]*

> "Every PII column across every source. Which ones are being extracted, which are metadata-only. Confidence scores. Source and table context. Click Export — you have your SOC2 package."

*[Click Export SOC2 Package button]*

> "Clean audit. No compliance holds on expansion. Five more systems connected that quarter."

---

## Closing (2 minutes) — The ask

> "What you just saw is Phase 1 — built on existing Fivetran Platform Connector capabilities and dbt. No new infrastructure. Shipped in three months."

> "The full vision has three phases:"

*[Can pull up the roadmap slide or just talk through it]*

> "Phase 2: first-class metadata-only mode in Fivetran, real-time change detection, the Fusion-to-Fivetran API bridge.
> Phase 3: the full Decision Engine — demand-driven materialization, source-level contracts, semantic validation, the autonomous governance loop."

> "The window is twelve months. Databricks is building toward this from the warehouse side. Snowflake is building from the storage side. Neither controls both ingestion **and** transformation. We do. Heimdall is how we turn that into a product story."

*[pause]*

> "Questions?"

---

## Handling tough questions

**"How is this different from Alation/Collibra?"**
> "Traditional catalogs document data **after** it lands in the warehouse. They answer 'what do we have?' Heimdall answers 'what **should** we bring in?' — and enforces that decision. It's also not a standalone tool — it's embedded in dbt and Fivetran. No new UI, no new deployment."

**"Can't dbt already do schema contracts?"**
> "dbt contracts today enforce schema at the model level, after transformation. Heimdall extends that to **source systems** — pre-ingestion, before data enters the warehouse. That's genuinely new capability."

**"What's the revenue model?"**
> "Free metadata-only connections remove the adoption friction that's blocking connector expansion today. The average customer has 15 of 60 sources connected. Heimdall makes it safe to connect everything — and once teams discover data through the AI catalog, they activate extraction. Metadata-only is the top of a conversion funnel. We project 3x increase in average source connections with 30-40% converting to paid sync within 12 months."

**"What if Fivetran engineering is slower than expected?"**
> "Phase 1 requires zero Fivetran changes. It runs entirely on the existing Platform Connector. That buys us three months to prove the concept, build the community, and get design partners — before we need any cross-team coordination."

**"This needs a real Anthropic API key for PII classification — is that a dependency?"**
> "For the AI features, yes. But the governance dashboard, change detection, policy engine, and audit trail all work without AI. The AI layer is additive — it makes governance automatic instead of manual. The core value proposition works on day one."

---

## Reset the demo

If you need to reset change events before a new audience:

```bash
# Clear change events so the feed looks fresh
psql postgres://heimdall@localhost:5432/heimdall \
  -c "DELETE FROM schema_change_events;"

# The worker will fire new demo events within 30s automatically
```

---

## Live URLs during demo

| What | URL |
|---|---|
| Control Center | http://localhost:3000 |
| Source Catalog | http://localhost:3000/catalog |
| Change Detection | http://localhost:3000/changes |
| Policy Engine | http://localhost:3000/policies |
| Compliance Audit | http://localhost:3000/audit |
| API health | http://localhost:8080/api/dashboard/stats |
