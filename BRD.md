# Business Requirements Document
## Learning Management System (LMS) API — Nigerian Primary & Secondary Schools

**Version:** 1.0.0
**Status:** Draft
**Prepared by:** Platform Architecture Team
**Date:** 2026-03-18

---

## Table of Contents

1. [Executive Summary](#1-executive-summary)
2. [Business Context](#2-business-context)
3. [Stakeholders & User Roles](#3-stakeholders--user-roles)
4. [System Architecture Overview](#4-system-architecture-overview)
5. [Core Domain Model](#5-core-domain-model)
6. [Functional Requirements](#6-functional-requirements)
   - 6.1 [Tenant Management](#61-tenant-management)
   - 6.2 [Academic Structure](#62-academic-structure)
   - 6.3 [User Management](#63-user-management)
   - 6.4 [Timetable Management](#64-timetable-management)
   - 6.5 [Assessments](#65-assessments)
   - 6.6 [Examinations](#66-examinations)
   - 6.7 [Student Progress & Reporting](#67-student-progress--reporting)
   - 6.8 [Virtual Meetings](#68-virtual-meetings)
   - 6.9 [Communications](#69-communications)
   - 6.10 [Notifications](#610-notifications)
   - 6.11 [Subscription & Billing](#611-subscription--billing)
   - 6.12 [Audit & Compliance](#612-audit--compliance)
7. [Non-Functional Requirements](#7-non-functional-requirements)
8. [API Design Principles](#8-api-design-principles)
9. [Integration Requirements](#9-integration-requirements)
10. [Security Requirements](#10-security-requirements)
11. [Business Rules Summary](#11-business-rules-summary)
12. [Glossary](#12-glossary)

---

## 1. Executive Summary

This document defines the business requirements for a multi-tenant Learning Management System (LMS) API designed specifically for Nigerian primary and secondary schools. The platform enables schools (tenants) to manage their full academic lifecycle — from session and term configuration to student assessments, timetabling, and communications — through a structured, role-based API.

The system is SaaS-based, operates on a postpaid subscription model, and is billed at **NGN 500 per student per term**, calculated automatically at the start of each term based on enrolled student count.

---

## 2. Business Context

### 2.1 Problem Statement

Nigerian schools — both primary and secondary — lack affordable, locally-aware digital infrastructure for managing academic operations. Existing global LMS platforms do not account for the Nigerian academic calendar structure (Sessions and Terms), local public holidays, multi-school group ownership, or NGN-denominated billing.

### 2.2 Opportunity

By building a multi-tenant API-first LMS tailored to the Nigerian school system, the platform can serve individual schools, school groups (conglomerates), and government-affiliated institutions through a single scalable infrastructure.

### 2.3 Business Goals

- Provide schools with a self-contained digital academic management system.
- Enable platform operators to onboard and manage multiple school tenants independently.
- Generate sustainable recurring revenue through a transparent per-student postpaid billing model.
- Reduce administrative overhead for school operators through automation (timetabling, billing, notifications).

---

## 3. Stakeholders & User Roles

### 3.1 Role Hierarchy

```
SUPER_ADMIN  (Platform Owner)
    └── ADMIN        (Tenant / School Manager)
            ├── TUTOR    (Teacher / Instructor)
            └── STUDENT  (Learner)
```

### 3.2 Role Definitions

#### SUPER_ADMIN
The platform owner. Operates at the infrastructure level above all tenants.

**Responsibilities:**
- Onboard and offboard tenants (schools)
- Configure platform-wide settings (base pricing, supported features)
- View aggregated billing and usage across all tenants
- Suspend or reactivate tenants
- Access audit logs across all tenants
- Manage RBAC and permissions (permissions will be an array of resource:action strings, e.g., "tenant:create", "admin:delete", etc)
- Manage platform-level integrations (payment gateway, email provider, video conferencing)

#### ADMIN
The manager of a single tenant (school). Has full control within their tenant's boundary.

**Responsibilities:**
- Configure and manage Sessions and Terms
- Define Term start/end dates and public holidays
- Manage Classes and Course assignments
- Enroll and manage Students and Tutors
- View and manage billing and student subscription counts
- Send communications to all users within the tenant
- Approve or reject Timetable swap requests escalated beyond the Tutor level
- Generate school-wide reports

#### TUTOR
A teacher assigned to one or more Courses within one or more Classes.

**Responsibilities:**
- Manage Courses they are assigned to
- Create and publish Quizzes, Assignments, and Examinations
- Grade submissions and provide feedback
- View student progress within their assigned Courses
- Schedule and host virtual Meetings for their assigned Classes
- Send communications to their assigned Classes and Courses
- Request Timetable period swaps with other Tutors
- Approve or reject incoming Timetable swap requests

#### STUDENT
An enrolled learner assigned to a Class.

**Responsibilities:**
- Access Courses assigned to their Class
- Submit Quizzes and Assignments
- Sit Examinations within configured windows
- View their own progress and grades
- Receive communications and notifications
- Join virtual Meetings for their Class

---

## 4. System Architecture Overview

### 4.1 Multi-Tenancy Model

The platform uses **logical multi-tenancy**. Each tenant (school) is fully isolated at the data level via a `tenant_id` scoped to every resource. The SUPER_ADMIN operates outside tenant scope.

### 4.2 Authentication & Authorization

- All API access is authenticated via JWT tokens.
- Tokens carry `role`, `tenant_id`, and `user_id` claims.
- Role-based access control (RBAC) is enforced at the API gateway and service level.
- SUPER_ADMIN tokens carry no `tenant_id` and can access any tenant's data with explicit scope.
- SUPARA_ADMIN and ADMIN must be created with a permissions field (permissions will be an array of resource:action strings, e.g., "tenant:create", "admin:delete", etc)

### 4.3 Service Boundaries (Suggested Decomposition)

| Service | Responsibility |
|---|---|
| Identity Service | Auth, users, roles, sessions |
| Academic Service | Sessions, Terms, Classes, Courses, Timetables |
| Assessment Service | Quizzes, Assignments, Examinations, Submissions |
| Progress Service | Student progress, grades, reports |
| Communication Service | Emails, notifications, announcement routing |
| Meeting Service | Virtual meeting scheduling and session management |
| Billing Service | Student counts, term invoicing, payment tracking |
| Audit Service | Event logging, compliance records |

---

## 5. Core Domain Model

### 5.1 Entity Overview

```
TENANT
  └── SESSION (School Year)
        └── TERM (3 per Session)
              ├── CLASS
              │     ├── COURSE
              │     │     ├── TUTOR (assigned)
              │     │     ├── QUIZ
              │     │     ├── ASSIGNMENT
              │     │     └── EXAMINATION
              │     ├── TIMETABLE
              │     │     └── PERIOD
              │     └── MEETING
              └── STUDENT
                    └── PROGRESS (per Course per Term)
```

### 5.2 Key Entity Descriptions

| Entity | Description |
|---|---|
| `Tenant` | A school or school group. The root isolation boundary. |
| `Session` | An academic year (e.g., 2025/2026). Scoped per tenant. |
| `Term` | One of three academic periods within a Session. Has a configured start date, end date, and holiday calendar. |
| `Class` | A student cohort group (e.g., JSS1A, Primary 4B). Belongs to a Session. |
| `Course` | A subject taught within a Class (e.g., Mathematics, English). Has exactly one assigned Tutor per Term. |
| `Tutor` | A user with the TUTOR role. Can be assigned to multiple Courses across multiple Classes. |
| `Student` | A user with the STUDENT role. Enrolled in exactly one Class per Session. |
| `Timetable` | Auto-generated schedule of Periods for a Class within a Term. |
| `Period` | A single scheduled block in the Timetable (day, start time, end time, course, tutor). |
| `Quiz` | A formative assessment created by a Tutor for a Course. |
| `Assignment` | A task-based assessment with a submission deadline. |
| `Examination` | A formal end-of-term assessment with a strict time window. |
| `Progress` | A record of a Student's academic standing per Course per Term (grades, completion, attendance). |
| `Meeting` | A virtual call session created by a Tutor for a Class. |
| `Subscription` | A tenant's billing record per Term, calculated from enrolled Student count. |

---

## 6. Functional Requirements

### 6.1 Tenant Management

#### FR-TEN-001: Tenant Onboarding
The SUPER_ADMIN must be able to create a new tenant with the following attributes: school name, school type (Primary, Secondary, or Combined), contact email, address, logo, and billing contact information. Tenanr must be onbaord with a principal ADMIN, who has full permission to every resource and action within a TENANT

#### FR-TEN-002: Tenant Configuration
Each tenant must maintain its own configuration namespace covering: default timezone (WAT), academic level type, supported Classes, branding assets, and communication preferences.

#### FR-TEN-003: Tenant Suspension
The SUPER_ADMIN must be able to suspend a tenant for non-payment or policy violations. Suspended tenants must have API access revoked while data is preserved.

#### FR-TEN-004: Tenant Reactivation
The SUPER_ADMIN must be able to reactivate a suspended tenant. Upon reactivation, all previously active users and configurations must be restored.

---

### 6.2 Academic Structure

#### FR-ACA-001: Session Creation
The ADMIN must be able to create a Session (school year) for their tenant specifying a label (e.g., "2025/2026") and status (active, archived). Only one Session can be active at a time per tenant.

#### FR-ACA-002: Term Configuration
Each Session must have exactly three Terms. The ADMIN must configure each Term with: an ordinal (1st, 2nd, 3rd), start date, end date, and a list of public holidays and school-specific non-instructional days. Terms within a Session must not overlap.

#### FR-ACA-003: Class Management
The ADMIN must be able to create Classes within a Session. A Class must have a name (e.g., JSS1, Primary 5), an arm/stream (e.g., A, B, Gold), and a school level (Primary or Secondary). Classes persist across Terms within the same Session.

#### FR-ACA-004: Course Management
The ADMIN must be able to create Courses and assign them to a Class within a Session. Each Course must have a name, a subject code, and an assigned Tutor per Term. A Tutor can be assigned to the same Course across multiple Classes.

#### FR-ACA-005: Tutor Course Reassignment
The ADMIN must be able to reassign a Course to a different Tutor between Terms. Historical grade data must remain attributed to the original Tutor.

#### FR-ACA-006: Student Enrollment
The ADMIN must be able to enroll Students into a Class for an active Session. A Student can only be enrolled in one Class per Session. Transfer between Classes within a Session must be supported with an audit trail.

---

### 6.3 User Management

#### FR-USR-001: User Invitation
The ADMIN must be able to invite Tutors and Students to the platform via email. Invitations must include a role, a tenant context, and an expiring acceptance link.

#### FR-USR-002: Profile Management
All users must be able to manage their own profile (name, profile photo, password, contact details). ADMIN can update any user's profile within their tenant.

#### FR-USR-003: User Deactivation
The ADMIN must be able to deactivate a user within their tenant without deleting their data. Deactivated users must lose API access immediately.

#### FR-USR-004: Role Management
The SUPER_ADMIN must be able to promote a TUTOR to an ADMIN within a tenant. An ADMIN can demote another ADMIN to TUTOR within the same tenant.

#### FR-USR-005: Password Reset
All users must be able to request a password reset via email. Reset links must expire after 30 minutes.

---

### 6.4 Timetable Management

#### FR-TTB-001: Automatic Timetable Generation
The system must automatically generate a Timetable for each Class at the start of each Term. The generation algorithm must:
- Distribute all Courses across the available instructional days of the Term.
- Ensure no Tutor is scheduled in two Periods at the same time across different Classes.
- Ensure no Class has two Courses scheduled simultaneously.
- Respect configurable Period durations (default: 40 minutes) and daily Period limits.
- Exclude public holidays and non-instructional days configured by the ADMIN.
- Produce a balanced weekly schedule with no Course exceeding a configurable maximum periods-per-week.

#### FR-TTB-002: Timetable Publication
A generated Timetable must remain in `DRAFT` status until explicitly published by the ADMIN. Published Timetables are visible to Tutors and Students.

#### FR-TTB-003: Period Swap Request
A Tutor must be able to request a swap of one of their Periods with a Period belonging to another Tutor within the same Class on the same day. A swap request must specify: the requesting Tutor's Period, the target Tutor's Period, and an optional reason.

#### FR-TTB-004: Swap Approval by Target Tutor
The target Tutor must receive a notification of the swap request and must be able to approve or reject it. An approved swap takes effect immediately on the published Timetable.

#### FR-TTB-005: Swap Escalation
If the target Tutor rejects a swap, the requesting Tutor may escalate to the ADMIN. The ADMIN can override and force the swap with a documented reason.

#### FR-TTB-006: Timetable Regeneration
The ADMIN must be able to trigger a full or partial Timetable regeneration for a Class without affecting other Classes. Regeneration invalidates existing swap requests for that Class.

#### FR-TTB-007: Timetable Viewing
Tutors must see a Timetable filtered to their assigned Classes and Courses. Students must see only the Timetable for their enrolled Class.

---

### 6.5 Assessments

#### FR-ASS-001: Quiz Creation
A Tutor must be able to create a Quiz for a Course they are assigned to, specifying: title, instructions, availability window (start and end datetime), time limit, total marks, and whether it is visible to students before the window opens.

#### FR-ASS-002: Quiz Questions
A Quiz must support the following question types: multiple choice (single answer), multiple choice (multiple answers), true/false, and short answer. Each question must have a mark value and an optional explanation shown after submission.

#### FR-ASS-003: Quiz Submission
Students must be able to attempt a Quiz once within the availability window. The system must enforce the time limit and auto-submit on expiry. A Student cannot retake a Quiz unless the Tutor explicitly resets their attempt.

#### FR-ASS-004: Quiz Grading
Multiple choice, multiple answer, and true/false questions must be auto-graded. Short answer questions must be flagged for manual Tutor review and grading.

#### FR-ASS-005: Assignment Creation
A Tutor must be able to create an Assignment for a Course, specifying: title, description, attachments (file uploads), submission deadline, maximum marks, and allowed submission formats.

#### FR-ASS-006: Assignment Submission
Students must be able to submit an Assignment before the deadline by uploading one or more files. Late submissions must be flagged but allowed unless the Tutor configures a hard cutoff.

#### FR-ASS-007: Assignment Grading
The Tutor must be able to review each submission, assign a mark, and leave written feedback. Graded results must be reflected in the Student's Progress record immediately.

#### FR-ASS-008: Assessment Visibility
Students may only view their own submissions and grades. Tutors may view all submissions within their assigned Courses. ADMINs may view all assessment data within their tenant.

---

### 6.6 Examinations

#### FR-EXM-001: Examination Creation
The ADMIN or Tutor (with ADMIN permission) must be able to create an Examination for a Course at the end of each Term, specifying: title, examination window (start and end datetime), duration, total marks, and instructions.

#### FR-EXM-002: Examination Window Enforcement
An Examination must only be accessible to Students within the configured window. The system must enforce the duration and auto-submit on expiry. No access is permitted outside the window.

#### FR-EXM-003: Examination Questions
Examinations must support all question types available for Quizzes. Questions may be marked confidential, preventing them from appearing in Student review after grading.

#### FR-EXM-004: Examination Grading
Auto-gradable questions must be scored immediately on submission. Subjective questions must be queued for Tutor grading. Final Examination scores must be locked by the Tutor before being visible to Students.

#### FR-EXM-005: Result Publication
The Tutor must explicitly publish Examination results. Published results are visible to the Student and reflected in their Term Progress summary. The ADMIN can override and publish results if a Tutor is unavailable.

#### FR-EXM-006: Examination Integrity
The system must record the Student's IP address, submission timestamp, and any detected tab-switch or focus-loss events during the Examination for Tutor and ADMIN review.

---

### 6.7 Student Progress & Reporting

#### FR-PRG-001: Progress Tracking
The system must maintain a Progress record for each Student per Course per Term, automatically aggregating: Quiz scores, Assignment scores, Examination score, and a computed overall grade.

#### FR-PRG-002: Grade Weighting
The ADMIN must be able to configure the grade weighting formula per tenant (e.g., Continuous Assessment = 40%, Examination = 60%). This weighting applies to all Courses unless overridden at the Course level.

#### FR-PRG-003: Student Report Card
At the end of each Term, the system must auto-generate a report card for each Student containing: all Course grades, class position (computed from all Students in the same Class), teacher remarks (entered by Tutor), and principal remarks (entered by ADMIN). Report cards must be exportable as PDF.

#### FR-PRG-004: Attendance Tracking
Tutors must be able to mark attendance for each Period. The system must compute a per-Course attendance percentage per Student per Term. Students below a configurable attendance threshold must be flagged.

#### FR-PRG-005: Progress Visibility
Students can view only their own Progress. Tutors can view Progress for Students in their assigned Courses. ADMINs can view all Progress within the tenant.

#### FR-PRG-006: Academic Standing Flags
The system must automatically flag Students whose overall grade falls below a configurable minimum threshold. Flagged Students must appear in a dedicated ADMIN and Tutor alert dashboard.

---

### 6.8 Virtual Meetings

#### FR-MTG-001: Meeting Scheduling
A Tutor must be able to schedule a virtual Meeting for a Class they are assigned to. A Meeting must have: title, description, scheduled start datetime, estimated duration, and a Class scope. The Tutor may not schedule a Meeting for a Class they are not assigned to.

#### FR-MTG-002: Meeting Access
All Students enrolled in the specified Class must automatically receive a Meeting invitation and join link. No manual RSVP is required unless configured by the ADMIN.

#### FR-MTG-003: Meeting Lifecycle
A Meeting must transition through the states: `SCHEDULED → LIVE → ENDED`. The Tutor starts and ends the Meeting. The system must record actual start time, end time, and participant join/leave events.

#### FR-MTG-004: Meeting Recordings
If the integrated conferencing provider supports it, Meeting recordings must be stored and linked to the Meeting record. Recordings are accessible to the Tutor and to Students in the Class for a configurable retention period.

#### FR-MTG-005: Meeting Cancellation
A Tutor must be able to cancel a scheduled Meeting. All enrolled Students and the ADMIN must receive a cancellation notification.

#### FR-MTG-006: ADMIN Meeting Oversight
The ADMIN must be able to view all scheduled, live, and past Meetings within the tenant. The ADMIN can cancel a Meeting with a documented reason.

---

### 6.9 Communications

#### FR-COM-001: ADMIN Broadcast Email
The ADMIN must be able to compose and send an email to one or more of the following recipient scopes: all Tutors in the tenant, all Students in the tenant, all users in a specific Class (Tutors and Students), or all users in the tenant.

#### FR-COM-002: Tutor Email
A Tutor must be able to compose and send an email to: all Students in a Class they are assigned to, all Students enrolled in a specific Course they teach, or all Tutors co-assigned to the same Class.

#### FR-COM-003: Email Delivery
All emails must be sent via the platform's configured transactional email provider. Sent emails must be logged with recipient list, sender, timestamp, and delivery status per recipient.

#### FR-COM-004: Email Scheduling
The ADMIN and Tutor must be able to schedule an email for future delivery. Scheduled emails must be cancellable before the send time.

#### FR-COM-005: Communication Restrictions
Tutors must not be able to send emails to Classes or Courses they are not assigned to. Students cannot send emails through the platform.

---

### 6.10 Notifications

#### FR-NOT-001: In-App Notifications
The system must deliver real-time in-app notifications to users for the following events: new Quiz or Assignment published, upcoming Assessment deadline (configurable reminder window), Examination window opening and closing, Timetable published or updated, Meeting scheduled or cancelled, Grade or report published, Subscription invoice generated, and any direct communication received.

#### FR-NOT-002: Push Notifications
The system must support push notifications to registered mobile devices for all in-app notification events.

#### FR-NOT-003: ADMIN Notification Broadcast
The ADMIN must be able to send a manual notification to any user scope within the tenant (same scopes as email in FR-COM-001).

#### FR-NOT-004: Tutor Notification Broadcast
The Tutor must be able to send a manual notification to Students in their assigned Classes and Courses.

#### FR-NOT-005: Notification Preferences
Users must be able to configure their notification preferences, toggling specific event types on or off for in-app and push channels independently.

#### FR-NOT-006: Notification History
All notifications must be persisted and accessible via API. Users must be able to mark notifications as read individually or all at once.

---

### 6.11 Subscription & Billing

#### FR-BIL-001: Postpaid Billing Model
The platform operates on a postpaid model. At the start of each Term, the system must automatically calculate the tenant's subscription charge as:

```
Invoice Amount (NGN) = Active Student Count × 500 × Number of Terms Remaining in Session
```

For per-term billing, the formula is:

```
Term Invoice (NGN) = Active Student Count at Term Start × NGN 500
```

#### FR-BIL-002: Student Count Snapshot
At the moment a Term transitions to `ACTIVE`, the system must snapshot the exact count of enrolled, active Students for that tenant. This count is immutable for that Term's invoice regardless of subsequent enrollment changes.

#### FR-BIL-003: Invoice Generation
The system must auto-generate an itemised invoice for each tenant at the start of each Term. The invoice must include: tenant name, Term label, snapshot student count, unit price (NGN 500), total amount, and due date.

#### FR-BIL-004: Invoice Delivery
Generated invoices must be delivered to the tenant's billing contact email and accessible in the ADMIN dashboard.

#### FR-BIL-005: Payment Tracking
The system must record payment status per invoice: `PENDING`, `PAID`, `OVERDUE`, or `DISPUTED`. Payment confirmation may be received via webhook from the integrated payment gateway.

#### FR-BIL-006: Overdue Handling
If a Term invoice remains unpaid beyond the configured grace period (default: 14 days), the tenant's account must be flagged as `PAYMENT_OVERDUE`. The SUPER_ADMIN must be alerted. Continued non-payment beyond a secondary threshold (default: 30 days) must trigger automatic suspension.

#### FR-BIL-007: Billing History
The ADMIN must be able to view all past invoices and payment history. The SUPER_ADMIN must be able to view billing history across all tenants.

#### FR-BIL-008: Manual Billing Adjustments
The SUPER_ADMIN must be able to apply credits, discounts, or manual adjustments to a tenant's invoice with a documented reason.

---

### 6.12 Audit & Compliance

#### FR-AUD-001: Audit Log
Every state-changing action in the system must produce an immutable audit log entry containing: actor user ID and role, tenant ID, action type, affected resource type and ID, before/after state snapshot, timestamp, and IP address.

#### FR-AUD-002: Audit Log Access
The SUPER_ADMIN can query audit logs across all tenants. The ADMIN can query audit logs scoped to their tenant only.

#### FR-AUD-003: Audit Log Retention
Audit logs must be retained for a minimum of 7 years in compliance with Nigerian data governance expectations.

#### FR-AUD-004: Sensitive Action Alerts
The system must alert the SUPER_ADMIN for the following sensitive actions: tenant suspension or reactivation, billing adjustment, ADMIN role promotion or demotion, and bulk student enrollment or removal.

---

## 7. Non-Functional Requirements

### 7.1 Performance

| Metric | Target |
|---|---|
| API response time (p95) | < 300ms for read operations |
| API response time (p95) | < 600ms for write operations |
| Timetable generation time | < 10 seconds per Class |
| Report card generation time | < 5 seconds per Student |
| Notification delivery latency | < 3 seconds (in-app) |
| System availability | 99.9% uptime (excluding planned maintenance) |

### 7.2 Scalability

- The system must support at minimum 500 concurrent tenants at launch.
- Each tenant may have up to 5,000 active Students.
- The system must scale horizontally to support 10x baseline load.

### 7.3 Data Integrity

- All financial calculations (student counts, invoice amounts) must use integer arithmetic in NGN Kobo to avoid floating point errors.
- Database transactions must be ACID-compliant for all billing and enrollment operations.

### 7.4 Localisation

- The platform's default timezone is West Africa Time (WAT, UTC+1).
- All date inputs and outputs must carry explicit timezone information.
- The system must support Nigerian public holiday calendars configurable per tenant.
- All monetary values are denominated in NGN.

### 7.5 Accessibility

- API responses must be consistent and machine-readable, enabling client applications to meet WCAG 2.1 AA accessibility standards.

---

## 8. API Design Principles

### 8.1 General

- The API must be RESTful with consistent resource-oriented URL structures.
- All responses must use `application/json`.
- Pagination must be cursor-based for all list endpoints.
- All list endpoints must support filtering and sorting on relevant fields.

### 8.2 Versioning

- The API must be versioned via URL path prefix: `/api/v1/`.
- Breaking changes must increment the version. Non-breaking changes may be additive within a version.

### 8.3 Error Handling

All error responses must follow a consistent envelope:

```json
{
  "error": {
    "code": "TERM_OVERLAP",
    "message": "The term dates overlap with an existing term in this session.",
    "details": {}
  }
}
```

### 8.4 Idempotency

All mutating endpoints (POST, PUT, PATCH) must support an `Idempotency-Key` header to prevent duplicate operations, particularly for billing and enrollment actions.

### 8.5 Rate Limiting

- Default rate limit: 1,000 requests per minute per tenant API key.
- SUPER_ADMIN endpoints: 200 requests per minute.
- Burst capacity: 2x sustained rate for up to 30 seconds.

### 8.6 Webhooks

The system must support outbound webhooks for the following events: invoice generated, invoice paid, invoice overdue, tenant suspended, student enrolled, assessment graded, and examination result published. Each webhook payload must be HMAC-signed for verification.

---

## 9. Integration Requirements

### 9.1 Payment Gateway

- The system must integrate with at least one Nigerian payment gateway (e.g., Paystack or Flutterwave) for invoice payment processing and webhook-based payment confirmation.

### 9.2 Transactional Email

- The system must integrate with a transactional email provider (e.g., Resend, Mailgun, or SendGrid) for all outbound email communications.

### 9.3 Push Notifications

- The system must integrate with Firebase Cloud Messaging (FCM) or a compatible provider for push notification delivery to Android and iOS devices.

### 9.4 Virtual Meetings

- The system must integrate with a video conferencing API (e.g., Daily.co, Zoom API, or Jitsi Meet) to create, manage, and end virtual Meeting sessions programmatically.

### 9.5 File Storage

- All file uploads (Assignment submissions, profile photos, Course materials) must be stored in a cloud object storage service (e.g., AWS S3 or Cloudflare R2). The API must return pre-signed URLs for client-side uploads and secure access URLs for downloads.

### 9.6 PDF Generation

- The system must be capable of generating PDFs server-side for: student report cards, invoices, and timetable exports.

---

## 10. Security Requirements

### 10.1 Authentication

- All API endpoints must require a valid JWT access token except for: login, password reset request, and invitation acceptance.
- Access tokens must expire after 15 minutes. Refresh tokens must expire after 30 days.
- Refresh token rotation must be enforced on each use.

### 10.2 Authorisation

- All endpoints must enforce tenant-scoped access. A user belonging to Tenant A must never be able to read or modify data belonging to Tenant B.
- Role permissions must be enforced server-side and never derived solely from client-provided parameters.

### 10.3 Data Encryption

- All data at rest must be encrypted (AES-256 or equivalent).
- All data in transit must use TLS 1.2 or higher.
- Student PII (names, contact details, academic records) must be treated as sensitive data and excluded from general-purpose logging.

### 10.4 Input Validation

- All API inputs must be validated against strict schemas. Unexpected fields must be rejected or stripped.
- File uploads must be type-validated and scanned for malware before storage.

### 10.5 Examination Integrity

- Examination attempt tokens must be single-use and bound to the authenticated Student's session.
- Each answer submission within an Examination must be individually persisted, not batched, to prevent total loss on connection failure.

---

## 11. Business Rules Summary

| ID | Rule |
|---|---|
| BR-001 | A Session may have exactly 3 Terms. No more, no less. |
| BR-002 | Terms within a Session must not have overlapping date ranges. |
| BR-003 | A Student can be enrolled in only one Class per Session. |
| BR-004 | A Course within a Class must have exactly one Tutor assigned per Term. |
| BR-005 | A Tutor may only schedule Meetings for Classes they are assigned to. |
| BR-006 | A Tutor may only send communications to Classes and Courses they are assigned to. |
| BR-007 | Only one Session per tenant may be in `ACTIVE` status at any time. |
| BR-008 | A Timetable Period swap requires approval from the target Tutor or ADMIN escalation. |
| BR-009 | Student billing count is snapshotted at Term start and is immutable for that Term's invoice. |
| BR-010 | Invoice amount = Active Student Count × NGN 500 per Term. |
| BR-011 | Unpaid invoices beyond the grace period trigger `PAYMENT_OVERDUE` status. |
| BR-012 | Continued non-payment beyond the secondary threshold triggers automatic tenant suspension. |
| BR-013 | Examination windows are strictly enforced. Students cannot access an Examination outside its window. |
| BR-014 | Examination results must be explicitly published by a Tutor or ADMIN before they are visible to Students. |
| BR-015 | Audit logs are immutable and retained for a minimum of 7 years. |
| BR-016 | The Timetable generator must not produce any scheduling clash for Tutors or Classes. |
| BR-017 | Grade weighting formula is configurable at tenant level with optional Course-level override. |
| BR-018 | A Student's Quiz attempt cannot be retaken without explicit Tutor reset. |
| BR-019 | Report cards are auto-generated at Term end and require Tutor remarks before finalisation. |
| BR-020 | Suspended tenants retain all data but lose all API access. |

---

## 12. Glossary

| Term | Definition |
|---|---|
| **Tenant** | A school or school group onboarded to the platform as an isolated organisational unit. |
| **Session** | An academic year (e.g., 2025/2026). Contains exactly 3 Terms. |
| **Term** | One of three sequential academic periods within a Session. |
| **Class** | A named student cohort within a Session (e.g., JSS2B). |
| **Course** | A subject taught within a Class by an assigned Tutor (e.g., Physics). |
| **Period** | A single scheduled instructional block within a Timetable. |
| **Timetable** | The complete schedule of Periods for a Class within a Term. |
| **Quiz** | A short, time-limited formative assessment within a Course. |
| **Assignment** | A task-based assessment with a file submission deadline. |
| **Examination** | A formal, high-stakes assessment at the end of a Term. |
| **Progress** | A Student's aggregated academic record per Course per Term. |
| **Meeting** | A scheduled virtual class session hosted by a Tutor. |
| **Subscription** | A tenant's billing record per Term based on enrolled Student count. |
| **SUPER_ADMIN** | The platform owner role with cross-tenant authority. |
| **ADMIN** | The school manager role scoped to a single tenant. |
| **TUTOR** | A teacher role assigned to Courses within one or more Classes. |
| **STUDENT** | A learner role enrolled in one Class per Session. |
| **WAT** | West Africa Time (UTC+1), the platform's default timezone. |
| **NGN** | Nigerian Naira, the currency of all monetary values on the platform. |

---