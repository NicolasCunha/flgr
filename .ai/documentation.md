# Documentation Rules

## Overview

This document defines how an AI agent working on flgr should interact with the project's documentation: business requirements ([docs/business/requirements](../docs/business/requirements/README.md)) and Architecture Decision Records ([docs/architecture/adr](../docs/architecture/adr/README.md)).

Reading and referencing existing requirements and ADRs is always allowed and encouraged — the rules below are about *creating or changing* them.

## When a new Business Requirement is needed

Consider a new business requirement when a task:

- Introduces a new user-facing behavior or feature not covered by an existing requirement.
- Changes existing behavior in a way not covered by its current requirement.
- Introduces a new quality attribute or constraint (performance, security, scalability, etc.).

Use the template at [0000-business-requirements-template.md](../docs/business/requirements/0000-business-requirements-template.md).

## When a new ADR is needed

Consider a new ADR when a task requires:

- Choosing between two or more viable technical approaches (a library, a pattern, a service, etc.).
- A decision that affects more than one part of the codebase, or would be costly to reverse.
- Deviating from, or extending, a decision already recorded in an existing ADR.

Use the template at [0000-adr-template.md](../docs/architecture/adr/0000-adr-template.md).

If a task fits entirely within the scope of an existing `Accepted` ADR, no new ADR is needed — reference the existing one instead.

## Mandatory confirmation before creating or changing a document

The agent may draft new business requirements and ADRs, and may propose changes to existing ones, but **must not create or edit the file until the user has explicitly confirmed it**.

Before creating a new document, the agent must:

1. Present the proposed title, type/category, and a short summary of the content directly to the user — not the full file contents.
2. Wait for explicit confirmation.
3. Only after confirmation, create the file from the appropriate template and update the corresponding `README.md` index table in the same directory.

The same applies to changing the `Status` of an existing ADR or requirement (e.g., marking one `Superseded` or `Deprecated`) — propose the change and wait for confirmation before editing, since `Status` determines what's authoritative for future work.

This applies even when the need for a document seems obvious from the task at hand. The agent must never silently add, remove, or modify entries in `docs/business/requirements` or `docs/architecture/adr` without the user's explicit go-ahead.
