# Agent Workflow

Steps an AI agent should follow when picking up a task in the flgr repository.

## 1. Ground the task in existing documentation

- Check [docs/business/requirements](../docs/business/requirements/README.md) for a business requirement that already covers the task.
- Check [docs/architecture/adr](../docs/architecture/adr/README.md) for ADRs governing the relevant area (see each ADR's `Category`).
- If the task isn't covered by either, see [documentation.md](documentation.md) for when a new document is warranted — and remember new documents always require the user's confirmation before being created.

## 2. Plan

- Break the task into steps.
- Identify which existing ADRs and requirements the implementation must respect.
- If the task seems to conflict with an existing `Accepted` ADR or requirement, do not silently override it — raise the conflict with the user before proceeding.

## 3. Implement

- Follow the decisions recorded in the applicable ADRs (e.g., stack and packaging decisions in [docs/architecture/adr](../docs/architecture/adr/README.md)).
- Coding conventions, folder structure, and testing strategy are not yet decided. Until they're recorded as ADRs, ask the user rather than assuming a convention.

## 4. Verify

- If the task is tied to a business requirement, check the implementation against that requirement's Acceptance Criteria.

## 5. Document

- If new business requirements or ADRs were confirmed and created per [documentation.md](documentation.md), make sure the corresponding `README.md` index table was updated.
