# Weeto PRD v1.3

**Status:** Ready for implementation  
**Tagline:** Schedule interviews, not messages.

---

## Problem Statement

Hiring teams in Iran coordinate interviews through Telegram, WhatsApp, phone calls, and email. A single interview often requires many back-and-forth messages: the recruiter asks for availability, the candidate replies, the recruiter suggests another time, a meeting link is created manually, reminders are sent by hand, and candidates still no-show or forget.

This process repeats dozens of times per week. Recruiters spend time scheduling instead of hiring. Candidates receive an unprofessional, fragmented experience. Missed interviews mean lost hiring opportunities.

Weeto replaces this manual coordination with a self-service booking flow: the recruiter sends one link, the candidate chooses a time in under 60 seconds, and confirmations, reminders, and meeting links are handled automatically.

**Initial target segment:** modern tech startups and hiring teams comfortable with Google Meet (VPN-friendly environments). Banks and organizations that cannot use VPN-backed services are supported later via Bale manual meeting links.

---

## Solution

Weeto is the hiring coordination layer for modern teams in Iran — not generic scheduling software.

Recruiters create an interview type with availability rules and receive a public booking link (e.g. `weeto.ir/{org}/{type}`). They paste that link into Telegram or any channel. Candidates open a beautiful, mobile-first, Persian-first booking page showing available times in the Jalali calendar. After booking, both parties receive email and SMS notifications. Google Meet links are created on the recruiter's own Google Calendar. Candidates can reschedule or cancel via a magic link — no account required.

The product makes hiring teams look professional by default, with cal.com-level simplicity and minimal configuration.

---

## User Stories

### Recruiter — account & organization

1. As a recruiter, I want to sign up with email and password, so that I can create a Weeto account without relying on Google.
2. As a recruiter, I want to sign up or log in with Google (Gmail), so that I can onboard quickly using an account I already have.
3. As a recruiter, I want to connect my Google account after signup, so that Weeto can create calendar events and Meet links on my behalf.
4. As a recruiter, I want to create an organization, so that my booking pages are branded under my company name.
5. As a recruiter, I want to see a clear prompt to connect Google when I create a Google Meet interview type without a connected account, so that I know what to do before sharing a link.

### Recruiter — availability & interview types

6. As a recruiter, I want to define my working hours, so that candidates only see times when I am available.
7. As a recruiter, I want to set interview duration and buffer time between interviews, so that I am not double-booked back-to-back.
8. As a recruiter, I want to set a maximum number of interviews per day, so that my schedule does not become overloaded.
9. As a recruiter, I want to block time off manually, so that personal appointments not on any calendar are respected.
10. As a recruiter, I want to create multiple interview types (e.g. "Frontend Interview", "Culture Fit"), so that I can share different links for different roles or stages.
11. As a recruiter, I want to choose a meeting provider per interview type (Google Meet, Bale link, or custom URL), so that I can support both tech teams and organizations that require native Iranian video services.
12. As a recruiter, I want to copy a public scheduling link for each interview type, so that I can paste it into Telegram, WhatsApp, or email in seconds.
13. As a recruiter, I want beautiful defaults with minimal configuration, so that I can start scheduling without learning a complex admin panel.

### Recruiter — interviews & dashboard

14. As a recruiter, I want to see a list of upcoming interviews, so that I know my schedule at a glance.
15. As a recruiter, I want to see today's interviews prominently, so that I can prepare for the day.
16. As a recruiter, I want to view candidate name, phone, and email for each booking, so that I have the information I need before the interview.
17. As a recruiter, I want to cancel an interview from the dashboard, so that I can free the slot when a interview is no longer happening.
18. As a recruiter, I want to receive a notification when a candidate books, reschedules, or cancels, so that I am never surprised by schedule changes.
19. As a recruiter, I want dates displayed in the Jalali calendar by default, so that the dashboard feels native and familiar.
20. As a recruiter, I want to optionally view dates in Gregorian, so that I can switch if I prefer.

### Candidate — booking (no account)

21. As a candidate, I want to open a booking link without creating an account, so that I can schedule quickly.
22. As a candidate, I want to see the company name, interview title, and available times, so that I understand what I am booking.
23. As a candidate, I want to see dates in the Jalali calendar with Persian day names, so that the page feels familiar and trustworthy.
24. As a candidate, I want to see the timezone (Asia/Tehran) clearly, so that I do not confuse interview times.
25. As a candidate, I want to enter my name, phone number, and email when booking, so that the recruiter can contact me.
26. As a candidate, I want to complete booking in under 60 seconds, so that the process is faster than messaging back and forth.
27. As a candidate, I want to see a confirmation page after booking, so that I know my interview is scheduled.
28. As a candidate, I want to receive an email confirmation with interview details, so that I have a record of the booking.
29. As a candidate, I want to receive an SMS confirmation, so that I have a reminder on the channel I actually check in Iran.
30. As a candidate, I want to receive a 24-hour SMS reminder before the interview, so that I am less likely to forget or no-show.
31. As a candidate, I want to reschedule my interview via a link in my SMS or email without logging in, so that I do not need to message the recruiter on Telegram.
32. As a candidate, I want to cancel my interview via a link in my SMS or email, so that the recruiter's slot is freed automatically.
33. As a candidate, I want reschedule and cancel links to stop working within a configurable window before the interview (default 4 hours), so that last-minute changes are handled directly with the recruiter.

### Notifications & meetings

34. As a recruiter, I want a Google Meet link created automatically when a candidate books a Google Meet interview type, so that I do not have to create links manually.
35. As a recruiter, I want the interview to appear on my Google Calendar, so that my schedule is in one place.
36. As a recruiter, I want Meet links created on my own Google account, so that I own the meeting and it appears on my calendar.
37. As a recruiter using Bale or a custom URL, I want to provide a meeting link template on the interview type, so that candidates receive the correct video link without Google.
38. As a recruiter, I want to receive an email when a candidate books, so that I am notified immediately.
39. As a candidate, I want the meeting link included in my confirmation and reminder messages, so that I can join without searching Telegram history.

### Plans & limits

40. As a free-plan recruiter, I want unlimited interviews per month, so that I am not forced back to Telegram when hiring volume increases.
41. As a free-plan recruiter, I want up to 15 Google Meet links per month, so that I can experience the core value before upgrading.
42. As a free-plan recruiter, I want up to 5 SMS messages per month, so that I can taste the no-show reduction benefit of SMS reminders.
43. As a free-plan recruiter, I want up to 3 interview types, so that I can cover my main hiring stages on the free tier.
44. As a pro-plan recruiter, I want unlimited Google Meet links, so that Meet auto-generation is not a bottleneck during active hiring.
45. As a pro-plan recruiter, I want 100 SMS messages per month included, so that reminders scale with my hiring volume.
46. As a pro-plan recruiter, I want to remove Weeto branding from my booking page, so that candidates see a professional company experience.
47. As a pro-plan recruiter, I want to see how many SMS credits I have used this month, so that I know when to upgrade or buy more.
48. As a business-plan recruiter, I want higher SMS limits and team scheduling features (future), so that Weeto becomes hiring infrastructure for my whole team.

### Design & UX

49. As any user, I want every screen to answer "what is the next action?", so that I never feel lost.
50. As any user, I want the product to work well on mobile, so that I can schedule from my phone.
51. As any user, I want the product to load fast, so that booking does not feel sluggish.
52. As any user, I want a Persian-first RTL interface, so that the product feels built for Iran.

### Go-to-market (operator stories)

53. As the Weeto team, we want 3 design partners who schedule at least one real interview through Weeto before public launch, so that we validate the product with real hiring workflows.
54. As the Weeto team, we want to document the hiring coordination problem through build-in-public content, so that we attract early adopters after MVP launch.

---

## Implementation Decisions

### Architecture overview

- **Frontend:** Next.js, React, TypeScript, Tailwind CSS, RTL, Jalali date display.
- **Backend:** Go standard library (`net/http`), no web framework.
- **Database:** PostgreSQL.
- **Schema & migrations:** SQL migration files (`goose` or `golang-migrate`) — source of truth for the database schema.
- **Go data access:** `sqlc` generates type-safe Go code from hand-written SQL queries; `pgx` as the PostgreSQL driver. No Prisma.
- **Background jobs:** Notification outbox table processed by a Go worker (Redis queue optional).
- **Auth:** Email/password signup and login; Google OAuth for Gmail signup and login.
- **Google integration:** Recruiter's OAuth refresh token (encrypted at rest) used for Google Calendar API event creation with Meet conference data.
- **SMS:** Kavenegar or Melipayamak (provider to be selected based on per-SMS cost and API quality).
- **Email:** SMTP or transactional email provider for confirmations and reminders.
- **Payments (v1.1):** Zarinpal one-time monthly payment activating Pro for 30 days; renewal reminder email on day 25. No complex subscription engine in v1.

### Database workflow (sqlc)

1. Write SQL migrations in `db/migrations/` (goose or golang-migrate).
2. Write SQL queries in `db/queries/` (one file per domain area).
3. Run `sqlc generate` to produce typed Go structs and query methods in `internal/db/`.
4. Handlers call generated sqlc methods via `pgx` connection pool — no ORM, no Prisma.

Schema changes flow: migration SQL → update query SQL → regenerate sqlc → update handlers.

### Primary integration seam

All product behavior is validated through the **public booking API seam**:

```
GET  /public/{orgSlug}/{typeSlug}/slots   → available slots
POST /public/{orgSlug}/{typeSlug}/book    → create booking (concurrency-safe)
```

Recruiter dashboard, notifications, Meet creation, and SMS all hang off a successful booking transaction. This is the single highest seam for end-to-end testing and vertical slice delivery.

Secondary seams:
- **Auth seam:** register, login, Google OAuth callback, token refresh.
- **Notification outbox seam:** enqueue on booking events; worker delivers email/SMS.

### Domain model (core entities)

- **User** — recruiter account; optional email/password; optional `google_id`; encrypted `google_refresh_token`.
- **Organization** — company workspace; slug for public URLs; plan tier and usage counters (`meet_links_used`, `sms_used`, `plan_expires_at`).
- **InterviewType** — title, slug, duration, buffer, meeting provider (`google_meet` | `bale_link` | `custom_url`), optional static meeting URL for non-Google providers.
- **AvailabilityRule** — working hours, breaks, max interviews per day, manual time-off blocks.
- **Slot** — generated bookable time window (`timestamptz` UTC); `booked` flag or derived from booking existence.
- **Booking** — candidate name, phone, email; links to slot and interview type; status (`scheduled` | `cancelled`); `meet_link`; Google Calendar event ID; signed `reschedule_token` and `cancel_token`.
- **NotificationOutbox** — event type, payload, status, retry count.

### Booking concurrency

Double-booking prevention is required on day one. Use a PostgreSQL transaction with row-level lock:

```
BEGIN → SELECT slot FOR UPDATE WHERE booked = false → INSERT booking → mark slot booked → COMMIT
```

Alternative: `UNIQUE(slot_id)` on bookings and handle duplicate key as 409 Conflict. Database is the lock — not in-process Go channels (which do not work across requests or server instances).

### Timezone & Jalali

- All timestamps stored as `timestamptz` in UTC.
- API returns ISO 8601.
- Frontend converts to `Asia/Tehran` and displays Jalali dates with Persian day names and numerals.
- Jalali conversion is a frontend concern; the API remains timezone-agnostic.

### Google OAuth dual purpose

A single Google OAuth connection serves both authentication and Calendar/Meet integration:

- On OAuth callback: create or find user, store encrypted refresh token.
- On booking (provider = `google_meet`): use recruiter's token to create Calendar event with `conferenceData` for Meet link; save `meet_link` and `calendar_event_id` on booking.
- Token refresh handled in worker or on 401 with short-lived access token cache in Redis.
- Interview types with `google_meet` require a connected Google account; show connect CTA otherwise. Email/password users can sign up first and connect Google as a second step.

### Meeting providers

| Provider | Behavior | Target segment |
|----------|----------|----------------|
| `google_meet` | Auto-created via recruiter's Google Calendar | Tech startups |
| `bale_link` | Recruiter provides Bale room URL on interview type | Banks, gov, no-VPN orgs |
| `custom_url` | Recruiter provides any meeting URL | Fallback |

Bale API auto-generation is out of scope for v1; manual URL field is sufficient.

### Reschedule & cancel (magic links)

- On booking confirmation, include signed tokens in SMS and email.
- Reschedule link opens the public booking page with candidate details pre-filled from token; shows new available slots; on confirm, frees old slot and books new one; notifies recruiter.
- Cancel link sets booking status to `cancelled` and frees slot.
- Reschedule/cancel disabled within X hours of interview start (default 4h, recruiter-configurable per org or interview type).

### Notification flow

On booking, reschedule, or cancel:
1. Write rows to `notification_outbox` within the booking transaction.
2. Worker processes outbox: send email to recruiter and candidate; send SMS where plan allows.
3. Scheduled reminders: 24-hour SMS reminder enqueued at booking time for future delivery.

SMS is plan-limited. Email is unlimited on all plans.

### Slot generation

- Generate slots from availability rules for a rolling window (e.g. 14 days ahead).
- Respect duration, buffer, max-per-day, and manual time-off.
- Google Calendar busy-block (read-only) is v1.1, not v1 launch.

### API surface (MVP)

**Auth**
- `POST /auth/register`
- `POST /auth/login`
- `GET  /auth/google` (redirect)
- `GET  /auth/google/callback`

**Recruiter (authenticated)**
- `POST /organizations`
- `GET  /organizations/:id`
- `POST /interview-types`
- `GET  /interview-types`
- `PUT  /interview-types/:id`
- `PUT  /availability`
- `GET  /bookings`
- `DELETE /bookings/:id` (cancel)

**Public**
- `GET  /public/:orgSlug/:typeSlug` (metadata for booking page)
- `GET  /public/:orgSlug/:typeSlug/slots`
- `POST /public/:orgSlug/:typeSlug/book`
- `GET  /public/reschedule/:token`
- `POST /public/reschedule/:token`
- `GET  /public/cancel/:token`
- `POST /public/cancel/:token`

### Build order (solo developer)

| Week | Deliverable |
|------|-------------|
| 1 | SQL migrations, sqlc setup, Go HTTP server, auth (email + Google OAuth), org + interview type CRUD, slot generation |
| 2 | Public slot listing and booking with concurrency lock; booking list for recruiter |
| 3 | Google Calendar/Meet on book; email notifications via outbox worker |
| 4 | SMS (confirm + 24h reminder); reschedule/cancel magic links |
| 5 | Next.js booking page (Jalali, RTL) and recruiter dashboard shell |

Week 2 milestone (curl/Postman demo): recruiter signs up, creates interview type, candidate books a slot, booking row exists, concurrent double-book returns 409.

### Pricing tiers

| Feature | Free | Pro (249k toman/mo) | Business (499k toman/mo) |
|---------|------|---------------------|--------------------------|
| Recruiters | 1 | Unlimited | Unlimited |
| Interviews/month | Unlimited | Unlimited | Unlimited |
| Interview types | 3 | Unlimited | Unlimited |
| Google Meet links | 15/month | Unlimited | Unlimited |
| SMS | 5/month | 100/month | 500/month |
| SMS overage | — | 150 toman each | Bulk rate |
| Email reminders | Unlimited | Unlimited | Unlimited |
| Weeto branding | Visible | Removable | Removable |
| Google Calendar busy-block | — | v1.1 | v1.1 + team |
| Custom domain | — | v1.1 | Yes |
| Round-robin | — | — | v2 |

Billing v1: manual Pro activation for design partners. Billing v1.1: Zarinpal payment link → 30-day Pro activation.

### Success metrics

- **North star:** Every interview scheduled in under 60 seconds with zero back-and-forth messages.
- **Month 1:** 3 design partners with ≥1 real booking; 10 companies total.
- **Month 3:** 30 companies, 1,000 interviews scheduled.
- **Month 6:** 100 paying companies.

---

## Testing Decisions

### Principles

- Test external behavior, not implementation details.
- Prioritize the public booking seam as the critical path — if booking works correctly under concurrency, the product works.
- Use integration tests against a real PostgreSQL test database for booking transactions and slot locking.
- Mock external services (Google Calendar API, SMS provider, email) at the HTTP boundary, not inside business logic.

### Modules to test

1. **Slot generation** — given availability rules, correct slots are produced respecting duration, buffer, max-per-day, and time-off.
2. **Booking creation** — successful book returns 201 with booking details; slot marked unavailable.
3. **Booking concurrency** — two simultaneous requests for the same slot: exactly one succeeds, the other receives 409.
4. **Reschedule flow** — valid token moves booking to new slot, frees old slot, invalid/expired token returns error.
5. **Cancel flow** — valid token cancels booking and frees slot.
6. **Plan limits** — free plan blocks Meet creation after 15 links; SMS not sent when monthly cap exceeded.
7. **Google Meet provider** — booking with `google_meet` provider and connected recruiter creates calendar event (mocked); booking fails gracefully when Google not connected.
8. **Notification outbox** — booking event enqueues correct notification rows; worker processes and marks sent.

### Prior art

Greenfield project — no existing test patterns. Establish integration test setup with test database and HTTP handler tests using Go's `httptest` package. SQL migrations run against test DB in CI before test suite; sqlc-generated queries used in all data access tests.

---

## Out of Scope

### v1 launch

- Team invites and multi-recruiter scheduling
- Round-robin interviewer assignment
- Google Calendar read-sync (busy blocking) — deferred to v1.1
- Outlook Calendar sync
- Analytics dashboard
- Custom domain
- Candidate CRM / pipeline / statuses beyond scheduled and cancelled
- Telegram bot integration
- Bale API auto-generation (manual URL only in v1)
- Integrated billing and Zarinpal payments — deferred to v1.1
- Full ATS, resume parsing, AI interview scoring
- Video interviewing within Weeto
- Payroll, HR management, performance reviews
- Enterprise SSO
- Candidate accounts or login
- Jalali/Gregorian toggle (Jalali default only in v1; toggle is nice-to-have)

### Future roadmap

- **v1.1:** Google Calendar busy-block, Zarinpal Pro billing, custom domain
- **v2:** Team scheduling, round-robin, candidate self-service polish, company branding, Outlook sync
- **v3:** Candidate scorecards, interview feedback, light hiring pipeline, offer scheduling, recruiter analytics
- **v4:** Recruiting operating system — ATS, recruiting CRM, referral management, talent pools

---

## Further Notes

### Positioning

Do not position Weeto as "a scheduling tool." Position it as **the hiring coordination layer for modern teams in Iran.** Scheduling is the entry point; the value is eliminating Telegram back-and-forth, reducing no-shows via SMS, and making teams look professional by default.

### Competitive context

Weeto does not compete with Calendly on features. It replaces the actual workflow: Telegram + WhatsApp + manual Meet links + missed reminders. Competitive advantages:

- Persian-first, Jalali calendar, RTL
- SMS reminders (critical in Iran)
- One-link paste into Telegram workflow
- Interview context on booking page (role, company, stage)
- Magic-link reschedule without candidate account

### Design principles

- Every screen answers: "What is the next action?"
- No complex admin panels
- Beautiful defaults, less configuration
- Mobile-first, Persian-first
- Fast loading, zero learning curve
- cal.com-level simplicity

### Go-to-market

1. **Before MVP:** Secure 3 design partners via direct Telegram outreach to recruiters and HR at tech startups. Each must schedule ≥1 real interview through Weeto.
2. **At launch:** Build-in-public YouTube content framed as personal story (problem discovery, not product demo title). First episode can be filmed before MVP exists — screen recording of messy Telegram scheduling.
3. **Month 2+:** Design partner testimonials, recruiter communities, continued build-in-public.

### Open decisions

- [ ] SMS provider: Kavenegar vs Melipayamak (compare per-SMS cost and API)
- [ ] Email provider: SMTP vs Resend or equivalent
- [ ] Exact SMS and Meet limit numbers after design partner feedback
- [ ] Reschedule cutoff default (4 hours proposed)

### Repository state

`weeto-frontend` is a greenfield Next.js starter (shadcn/ui, Tailwind). No backend, database schema, or product features exist yet. Backend will be a separate Go service in the monorepo or adjacent repository, using PostgreSQL with SQL migrations and sqlc-generated data access.
