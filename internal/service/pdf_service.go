package service

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"strings"
	"time"

	"arc-lms/internal/domain"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
)

// PDFService handles PDF generation
type PDFService struct {
	templatePath string
}

// NewPDFService creates a new PDF service
func NewPDFService(templatePath string) *PDFService {
	return &PDFService{
		templatePath: templatePath,
	}
}

// InvoicePDFData represents the data passed to the invoice PDF template
type InvoicePDFData struct {
	InvoiceNumber    string
	TenantName       string
	TenantAddress    string
	BillingEmail     string
	BillingPhone     string
	IssuedDate       string
	DueDate          string
	Status           string
	StatusClass      string
	PaidAt           string
	LineItems        []LineItemPDFData
	SubtotalFormatted string
	TaxAmount        int
	TaxFormatted     string
	DiscountAmount   int
	DiscountFormatted string
	TotalFormatted   string
	Notes            string
	GeneratedAt      string
}

// LineItemPDFData represents a line item in the invoice PDF
type LineItemPDFData struct {
	Description       string
	Quantity          int
	UnitPriceFormatted string
	AmountFormatted   string
}

// GenerateInvoicePDF generates a PDF for an invoice
func (s *PDFService) GenerateInvoicePDF(ctx context.Context, invoice *domain.Invoice, tenant *domain.Tenant) ([]byte, error) {
	// Prepare template data
	data := s.prepareInvoiceData(invoice, tenant)

	// Render HTML template
	html, err := s.renderInvoiceHTML(data)
	if err != nil {
		return nil, fmt.Errorf("failed to render invoice HTML: %w", err)
	}

	// Convert HTML to PDF
	pdf, err := s.htmlToPDF(ctx, html)
	if err != nil {
		return nil, fmt.Errorf("failed to convert HTML to PDF: %w", err)
	}

	return pdf, nil
}

// prepareInvoiceData prepares the data for the invoice template
func (s *PDFService) prepareInvoiceData(invoice *domain.Invoice, tenant *domain.Tenant) InvoicePDFData {
	data := InvoicePDFData{
		InvoiceNumber:     invoice.InvoiceNumber,
		TenantName:        tenant.Name,
		TenantAddress:     tenant.Address,
		BillingEmail:      invoice.BillingEmail,
		BillingPhone:      tenant.BillingContact.Phone,
		IssuedDate:        invoice.IssuedDate.Format("January 2, 2006"),
		DueDate:           invoice.DueDate.Format("January 2, 2006"),
		Status:            string(invoice.Status),
		StatusClass:       strings.ToLower(string(invoice.Status)),
		SubtotalFormatted: formatAmount(invoice.SubtotalAmount),
		TaxAmount:         invoice.TaxAmount,
		TaxFormatted:      formatAmount(invoice.TaxAmount),
		DiscountAmount:    invoice.DiscountAmount,
		DiscountFormatted: formatAmount(invoice.DiscountAmount),
		TotalFormatted:    formatAmount(invoice.TotalAmount),
		GeneratedAt:       time.Now().Format("January 2, 2006 at 3:04 PM"),
	}

	if invoice.PaidAt != nil {
		data.PaidAt = invoice.PaidAt.Format("January 2, 2006")
	}

	if invoice.Notes != nil {
		data.Notes = *invoice.Notes
	}

	// Convert line items
	for _, item := range invoice.LineItems {
		data.LineItems = append(data.LineItems, LineItemPDFData{
			Description:       item.Description,
			Quantity:          item.Quantity,
			UnitPriceFormatted: formatAmount(item.UnitPrice),
			AmountFormatted:   formatAmount(item.Amount),
		})
	}

	return data
}

// formatAmount formats an amount in Kobo to Naira with comma separators
func formatAmount(kobo int) string {
	naira := float64(kobo) / 100.0
	// Format with 2 decimal places and comma separators
	formatted := fmt.Sprintf("%.2f", naira)

	// Add comma separators
	parts := strings.Split(formatted, ".")
	intPart := parts[0]
	decPart := parts[1]

	// Add commas to integer part
	var result strings.Builder
	for i, digit := range intPart {
		if i > 0 && (len(intPart)-i)%3 == 0 {
			result.WriteRune(',')
		}
		result.WriteRune(digit)
	}

	return result.String() + "." + decPart
}

// renderInvoiceHTML renders the invoice HTML template
func (s *PDFService) renderInvoiceHTML(data InvoicePDFData) (string, error) {
	tmpl, err := template.ParseFiles(s.templatePath + "/invoice-pdf-template.html")
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

// htmlToPDF converts HTML content to PDF using headless Chrome
func (s *PDFService) htmlToPDF(ctx context.Context, html string) ([]byte, error) {
	// Create a new Chrome context
	allocCtx, cancel := chromedp.NewExecAllocator(ctx,
		append(chromedp.DefaultExecAllocatorOptions[:],
			chromedp.Flag("headless", true),
			chromedp.Flag("disable-gpu", true),
			chromedp.Flag("no-sandbox", true),
			chromedp.Flag("disable-dev-shm-usage", true),
		)...,
	)
	defer cancel()

	chromeCtx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	// Set a timeout for the PDF generation
	chromeCtx, cancel = context.WithTimeout(chromeCtx, 30*time.Second)
	defer cancel()

	var pdfBuf []byte

	// Navigate to the HTML content and generate PDF
	if err := chromedp.Run(chromeCtx,
		chromedp.Navigate("about:blank"),
		chromedp.ActionFunc(func(ctx context.Context) error {
			frameTree, err := page.GetFrameTree().Do(ctx)
			if err != nil {
				return err
			}
			return page.SetDocumentContent(frameTree.Frame.ID, html).Do(ctx)
		}),
		chromedp.ActionFunc(func(ctx context.Context) error {
			buf, _, err := page.PrintToPDF().
				WithPrintBackground(true).
				WithPaperWidth(8.5).
				WithPaperHeight(11).
				WithMarginTop(0.4).
				WithMarginBottom(0.4).
				WithMarginLeft(0.4).
				WithMarginRight(0.4).
				Do(ctx)
			if err != nil {
				return err
			}
			pdfBuf = buf
			return nil
		}),
	); err != nil {
		return nil, fmt.Errorf("failed to generate PDF: %w", err)
	}

	return pdfBuf, nil
}
