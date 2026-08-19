# Inngest Reference

Status: DRAFT

- **Original repository:** [inngest/inngest](https://github.com/inngest/inngest)
- **Default branch:** `main`
- **Pinned commit:** [`1f91829a35cccf2372768fef4aa275f56fbd4843`](https://github.com/inngest/inngest/commit/1f91829a35cccf2372768fef4aa275f56fbd4843)
- **Root license:** [Server Side Public License 1.0 with an Apache 2.0 future-license grant](https://github.com/inngest/inngest/blob/1f91829a35cccf2372768fef4aa275f56fbd4843/LICENSE.md)
- **Research status:** LICENSE REVIEWED; SOURCE ARCHITECTURE NOT YET ANALYZED

## License finding

At the pinned revision, the root `LICENSE.md` applies SSPL 1.0 and grants an additional Apache 2.0 license beginning on the third anniversary of the date each version is made available.

SSPL section 13 requires anyone making the program or a modified version available to third parties as a service to provide the defined Service Source Code under SSPL. That definition includes supporting management, interface, API, automation, monitoring, backup, storage, and hosting software used to provide the service.

This record is an engineering constraint, not legal advice. CompanyOS must obtain appropriate legal review before copying Inngest code, distributing a derivative, or operating it as part of a third-party service.

## CompanyOS disposition

- **Borrow now:** scheduling, cron/event-trigger, flow-control, and wake-on-demand concepts as patterns.
- **Do not borrow by default:** source code or an embedded/hosted Inngest service.
- **Dependency status:** NOT APPROVED; legal and architectural review required.
- **CompanyOS boundary:** Daemon and Runtime contracts remain CompanyOS-owned and provider-independent.
