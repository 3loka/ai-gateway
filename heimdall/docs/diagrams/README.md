# Heimdall — Architecture Diagrams

All diagrams are written in [Mermaid](https://mermaid.js.org/).
View in VS Code with the [Markdown Preview Mermaid Support](https://marketplace.visualstudio.com/items?itemName=bierner.markdown-mermaid) extension,
or open any `.md` file and use **Cmd+Shift+V** to preview.

| # | Diagram | Description |
|---|---------|-------------|
| [01](./01-system-context.md) | System Context | Users, Heimdall, and all external systems |
| [02](./02-component.md) | Component Diagram | Internal components: API, Core, Worker, DB |
| [03](./03-data-flow.md) | Data Flow | Phase A (metadata-only) → Phase B (policy-gated extraction) |
| [04](./04-change-detection-flow.md) | Change Detection Flow | How schema drift is detected, classified, and alerted |
| [05](./05-policy-evaluation-flow.md) | Policy Evaluation Flow | PII gate → contract → freshness SLA → cost guardrail → decision |
| [06](./06-sequence-day-zero.md) | Sequence: Day Zero | Connect 60 sources in metadata-only mode, AI classifies PII |
| [07](./07-sequence-breaking-change.md) | Sequence: Breaking Change | Salesforce field renamed → caught in 1 min → blast radius shown |
| [08](./08-erd.md) | Entity Relationship | Full database schema |
| [09](./09-state-machine.md) | State Machine | Source connection lifecycle: Disconnected → Crawling → Monitoring |
| [10](./10-deployment.md) | Deployment | Docker Compose setup, env vars, service topology |

## Key Flows for Leadership Demo

1. **The core idea** → [03 Data Flow](./03-data-flow.md) — Phase A vs Phase B
2. **The dramatic moment** → [07 Breaking Change](./07-sequence-breaking-change.md) — 10:01am alert
3. **The governance story** → [05 Policy Evaluation](./05-policy-evaluation-flow.md)
4. **The full lifecycle** → [09 State Machine](./09-state-machine.md)
