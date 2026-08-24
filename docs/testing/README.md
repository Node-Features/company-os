# Testing Documentation

This directory owns the strategy for verifying invariants, contracts, workflows, policies, resilience, and isolation.

| Document | Purpose | Status | Read when |
|---|---|---|---|
| `strategy.md` | Overall testing responsibilities and levels | APPROVED | Read when planning validation. |
| `contract-tests.md` | Shared-contract and adapter verification | APPROVED | Read when changing interfaces or adapters. |
| `failure-injection.md` | Crash, retry, replay, and recovery testing | APPROVED | Read when validating failure behavior. |
| `concurrency-guarantees.md` | Which mechanisms are exactly-once vs. at-least-once vs. best-effort, with the test proving each, plus the full 12-scenario concurrency test matrix | APPROVED | Read before adding a new CAS/fencing mechanism, or before claiming a delivery/execution guarantee that isn't already documented here. |
