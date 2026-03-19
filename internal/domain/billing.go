package domain

import (
	"time"

	"github.com/google/uuid"
)

// SubscriptionStatus represents the status of a tenant's subscription
type SubscriptionStatus string

const (
	SubscriptionStatusActive       SubscriptionStatus = "ACTIVE"
	SubscriptionStatusPaymentOverdue SubscriptionStatus = "PAYMENT_OVERDUE"
	SubscriptionStatusSuspended     SubscriptionStatus = "SUSPENDED"
	SubscriptionStatusCancelled     SubscriptionStatus = "CANCELLED"
)

// InvoiceStatus represents the payment status of an invoice
type InvoiceStatus string

const (
	InvoiceStatusPending  InvoiceStatus = "PENDING"
	InvoiceStatusPaid     InvoiceStatus = "PAID"
	InvoiceStatusOverdue  InvoiceStatus = "OVERDUE"
	InvoiceStatusDisputed InvoiceStatus = "DISPUTED"
	InvoiceStatusVoided   InvoiceStatus = "VOIDED"
)

// PaymentMethod represents the payment method used
type PaymentMethod string

const (
	PaymentMethodBankTransfer PaymentMethod = "BANK_TRANSFER"
	PaymentMethodCard         PaymentMethod = "CARD"
	PaymentMethodUSSD         PaymentMethod = "USSD"
	PaymentMethodManual       PaymentMethod = "MANUAL"
)

// Currency represents the currency code
type Currency string

const (
	CurrencyNGN Currency = "NGN" // Nigerian Naira
)

// Subscription represents a tenant's billing subscription
type Subscription struct {
	ID                uuid.UUID          `json:"id" validate:"required,uuid"`
	TenantID          uuid.UUID          `json:"tenant_id" validate:"required,uuid"`
	SessionID         uuid.UUID          `json:"session_id" validate:"required,uuid"`
	Status            SubscriptionStatus `json:"status" validate:"required,oneof=ACTIVE PAYMENT_OVERDUE SUSPENDED CANCELLED"`
	PricePerStudentPerTerm int           `json:"price_per_student_per_term" validate:"required,min=1"` // in Kobo (NGN * 100)
	Currency          Currency           `json:"currency" validate:"required,oneof=NGN"`
	StartDate         time.Time          `json:"start_date" validate:"required"`
	EndDate           *time.Time         `json:"end_date,omitempty"`
	CancelledAt       *time.Time         `json:"cancelled_at,omitempty"`
	CancellationReason *string           `json:"cancellation_reason,omitempty" validate:"omitempty,max=500"`
	CreatedAt         time.Time          `json:"created_at" validate:"required"`
	UpdatedAt         time.Time          `json:"updated_at" validate:"required"`
}

// IsActive returns true if the subscription is active
func (s *Subscription) IsActive() bool {
	return s.Status == SubscriptionStatusActive
}

// IsOverdue returns true if the subscription has overdue payments
func (s *Subscription) IsOverdue() bool {
	return s.Status == SubscriptionStatusPaymentOverdue
}

// IsSuspended returns true if the subscription is suspended
func (s *Subscription) IsSuspended() bool {
	return s.Status == SubscriptionStatusSuspended
}

// MarkAsOverdue marks the subscription as payment overdue
func (s *Subscription) MarkAsOverdue() {
	s.Status = SubscriptionStatusPaymentOverdue
	s.UpdatedAt = time.Now()
}

// Suspend marks the subscription as suspended
func (s *Subscription) Suspend() {
	s.Status = SubscriptionStatusSuspended
	s.UpdatedAt = time.Now()
}

// Reactivate marks the subscription as active
func (s *Subscription) Reactivate() {
	s.Status = SubscriptionStatusActive
	s.UpdatedAt = time.Now()
}

// Cancel marks the subscription as cancelled with a reason
func (s *Subscription) Cancel(reason string) {
	s.Status = SubscriptionStatusCancelled
	s.CancellationReason = &reason
	now := time.Now()
	s.CancelledAt = &now
	s.EndDate = &now
	s.UpdatedAt = now
}

// InvoiceLineItem represents a line item on an invoice
type InvoiceLineItem struct {
	Description string `json:"description" validate:"required,min=3,max=200"`
	Quantity    int    `json:"quantity" validate:"required,min=1"`
	UnitPrice   int    `json:"unit_price" validate:"required,min=0"` // in Kobo
	Amount      int    `json:"amount" validate:"required,min=0"`     // in Kobo (Quantity * UnitPrice)
}

// Invoice represents a billing invoice for a term
type Invoice struct {
	ID                  uuid.UUID         `json:"id" validate:"required,uuid"`
	TenantID            uuid.UUID         `json:"tenant_id" validate:"required,uuid"`
	SubscriptionID      uuid.UUID         `json:"subscription_id" validate:"required,uuid"`
	TermID              uuid.UUID         `json:"term_id" validate:"required,uuid"`
	InvoiceNumber       string            `json:"invoice_number" validate:"required,min=5,max=50"` // e.g., "INV-2025-001"
	Status              InvoiceStatus     `json:"status" validate:"required,oneof=PENDING PAID OVERDUE DISPUTED VOIDED"`
	Currency            Currency          `json:"currency" validate:"required,oneof=NGN"`
	LineItems           []InvoiceLineItem `json:"line_items" validate:"required,min=1,dive"`
	SubtotalAmount      int               `json:"subtotal_amount" validate:"required,min=0"` // in Kobo
	TaxAmount           int               `json:"tax_amount" validate:"min=0"`               // in Kobo
	DiscountAmount      int               `json:"discount_amount" validate:"min=0"`          // in Kobo
	TotalAmount         int               `json:"total_amount" validate:"required,min=0"`    // in Kobo
	StudentCount        int               `json:"student_count" validate:"required,min=0"`   // Snapshot at term start
	DueDate             time.Time         `json:"due_date" validate:"required"`
	IssuedDate          time.Time         `json:"issued_date" validate:"required"`
	PaidAt              *time.Time        `json:"paid_at,omitempty"`
	PaymentMethod       *PaymentMethod    `json:"payment_method,omitempty" validate:"omitempty,oneof=BANK_TRANSFER CARD USSD MANUAL"`
	PaymentReference    *string           `json:"payment_reference,omitempty" validate:"omitempty,max=100"`
	Notes               *string           `json:"notes,omitempty" validate:"omitempty,max=1000"`
	BillingEmail        string            `json:"billing_email" validate:"required,email"`
	PDFUrl              *string           `json:"pdf_url,omitempty" validate:"omitempty,url"`
	CreatedAt           time.Time         `json:"created_at" validate:"required"`
	UpdatedAt           time.Time         `json:"updated_at" validate:"required"`
	DisputedAt          *time.Time        `json:"disputed_at,omitempty"`
	DisputeReason       *string           `json:"dispute_reason,omitempty" validate:"omitempty,max=1000"`
	VoidedAt            *time.Time        `json:"voided_at,omitempty"`
	VoidReason          *string           `json:"void_reason,omitempty" validate:"omitempty,max=500"`
}

// IsPending returns true if the invoice is pending payment
func (i *Invoice) IsPending() bool {
	return i.Status == InvoiceStatusPending
}

// IsPaid returns true if the invoice has been paid
func (i *Invoice) IsPaid() bool {
	return i.Status == InvoiceStatusPaid
}

// IsOverdue returns true if the invoice is overdue
func (i *Invoice) IsOverdue() bool {
	return i.Status == InvoiceStatusOverdue
}

// IsDisputed returns true if the invoice is disputed
func (i *Invoice) IsDisputed() bool {
	return i.Status == InvoiceStatusDisputed
}

// IsVoided returns true if the invoice is voided
func (i *Invoice) IsVoided() bool {
	return i.Status == InvoiceStatusVoided
}

// MarkAsPaid marks the invoice as paid
func (i *Invoice) MarkAsPaid(paymentMethod PaymentMethod, reference string) {
	i.Status = InvoiceStatusPaid
	i.PaymentMethod = &paymentMethod
	i.PaymentReference = &reference
	now := time.Now()
	i.PaidAt = &now
	i.UpdatedAt = now
}

// MarkAsOverdue marks the invoice as overdue
func (i *Invoice) MarkAsOverdue() {
	i.Status = InvoiceStatusOverdue
	i.UpdatedAt = time.Now()
}

// Dispute marks the invoice as disputed with a reason
func (i *Invoice) Dispute(reason string) {
	i.Status = InvoiceStatusDisputed
	i.DisputeReason = &reason
	now := time.Now()
	i.DisputedAt = &now
	i.UpdatedAt = now
}

// Void marks the invoice as voided with a reason
func (i *Invoice) Void(reason string) {
	i.Status = InvoiceStatusVoided
	i.VoidReason = &reason
	now := time.Now()
	i.VoidedAt = &now
	i.UpdatedAt = now
}

// IsPaymentDue returns true if payment is due (past due date and not paid)
func (i *Invoice) IsPaymentDue(now time.Time) bool {
	return now.After(i.DueDate) && !i.IsPaid()
}

// DaysOverdue returns the number of days the invoice is overdue
func (i *Invoice) DaysOverdue(now time.Time) int {
	if !i.IsPaymentDue(now) {
		return 0
	}
	return int(now.Sub(i.DueDate).Hours() / 24)
}

// FormatAmountNGN returns the total amount formatted as NGN (converting from Kobo)
func (i *Invoice) FormatAmountNGN() float64 {
	return float64(i.TotalAmount) / 100.0
}

// BillingAdjustment represents manual adjustments to billing (credits, discounts, etc.)
type BillingAdjustment struct {
	ID          uuid.UUID `json:"id" validate:"required,uuid"`
	TenantID    uuid.UUID `json:"tenant_id" validate:"required,uuid"`
	InvoiceID   *uuid.UUID `json:"invoice_id,omitempty" validate:"omitempty,uuid"` // If adjustment is for specific invoice
	Type        string    `json:"type" validate:"required,oneof=CREDIT DISCOUNT CHARGE REFUND"`
	Amount      int       `json:"amount" validate:"required"` // in Kobo (can be negative)
	Currency    Currency  `json:"currency" validate:"required,oneof=NGN"`
	Reason      string    `json:"reason" validate:"required,min=10,max=1000"`
	AppliedBy   uuid.UUID `json:"applied_by" validate:"required,uuid"` // SUPER_ADMIN user ID
	CreatedAt   time.Time `json:"created_at" validate:"required"`
}
