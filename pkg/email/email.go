package email

import (
	"fmt"
	"time"

	"github.com/healthcare-market-research/backend/internal/config"
	"github.com/healthcare-market-research/backend/internal/domain/form"
	"github.com/healthcare-market-research/backend/internal/domain/order"
	gomail "gopkg.in/gomail.v2"
)

// EmailService defines the interface for sending email notifications.
type EmailService interface {
	SendFormNotification(submission *form.FormSubmission) error
	SendOrderConfirmation(o *order.Order) error
	SendOrderAdminNotification(o *order.Order) error
}

type smtpEmailService struct {
	cfg *config.EmailConfig
}

// NewSMTPEmailService creates an EmailService backed by an SMTP dialer.
func NewSMTPEmailService(cfg *config.EmailConfig) EmailService {
	return &smtpEmailService{cfg: cfg}
}

// SendFormNotification sends an HTML email notification for a form submission.
// It sends two emails: one to the admin (NotifyTo) and one confirmation to the client.
func (s *smtpEmailService) SendFormNotification(submission *form.FormSubmission) error {
	d := gomail.NewDialer(s.cfg.Host, s.cfg.Port, s.cfg.User, s.cfg.Password)

	// 1. Admin notification
	adminSubject, adminBody := buildEmail(submission)
	adminMsg := gomail.NewMessage()
	adminMsg.SetHeader("From", s.cfg.From)
	adminMsg.SetHeader("To", s.cfg.NotifyTo)
	adminMsg.SetHeader("Subject", adminSubject)
	adminMsg.SetBody("text/html", adminBody)

	if err := d.DialAndSend(adminMsg); err != nil {
		return err
	}

	// 2. Client confirmation
	clientEmail := strVal(submission.Data["email"])
	if clientEmail == "" {
		return nil
	}

	clientSubject, clientBody := buildClientConfirmationEmail(submission)
	clientMsg := gomail.NewMessage()
	clientMsg.SetHeader("From", s.cfg.From)
	clientMsg.SetHeader("To", clientEmail)
	clientMsg.SetHeader("Subject", clientSubject)
	clientMsg.SetBody("text/html", clientBody)

	return d.DialAndSend(clientMsg)
}

func buildClientConfirmationEmail(submission *form.FormSubmission) (subject, body string) {
	fullName := strVal(submission.Data["fullName"])

	if submission.Category == form.CategoryContact {
		subject = "We received your message — HealthcareForesights"
		body = fmt.Sprintf(`<!DOCTYPE html>
<html>
<body style="font-family:Arial,sans-serif;color:#333;max-width:600px;margin:0 auto;padding:20px">
  <h2 style="color:#1a73e8">Thank You for Contacting Us</h2>
  <p>Dear %s,</p>
  <p>We have received your message and our team will get back to you shortly.</p>
  <p>Your reference number is <strong>#%d</strong>. Please keep it for your records.</p>
  <p>If you have any urgent queries, feel free to reach us at <a href="mailto:support@healthcareforesights.com">support@healthcareforesights.com</a>.</p>
  <p>Thank you for reaching out to HealthcareForesights!</p>
</body>
</html>`, fullName, submission.ID)
	} else {
		reportTitle := strVal(submission.Data["reportTitle"])
		subject = "Sample Request Received — HealthcareForesights"
		body = fmt.Sprintf(`<!DOCTYPE html>
<html>
<body style="font-family:Arial,sans-serif;color:#333;max-width:600px;margin:0 auto;padding:20px">
  <h2 style="color:#1a73e8">Sample Request Received</h2>
  <p>Dear %s,</p>
  <p>Thank you for your interest in <strong>%s</strong>. We have received your sample request and will send the sample report to your email within 1–2 business days.</p>
  <p>Your reference number is <strong>#%d</strong>. Please keep it for your records.</p>
  <p>For any questions, contact us at <a href="mailto:support@healthcareforesights.com">support@healthcareforesights.com</a>.</p>
  <p>Thank you for choosing HealthcareForesights!</p>
</body>
</html>`, fullName, reportTitle, submission.ID)
	}

	return subject, body
}

func buildEmail(submission *form.FormSubmission) (subject, body string) {
	data := submission.Data
	fullName := strVal(data["fullName"])
	submittedAt := submission.CreatedAt.Format(time.RFC1123)
	submissionID := fmt.Sprintf("%d", submission.ID)
	metaRows := buildMetadataRows(submission.Metadata)

	if submission.Category == form.CategoryContact {
		subject = fmt.Sprintf("[Contact Form] New Submission – %s", fullName)
		body = fmt.Sprintf(`<!DOCTYPE html>
<html>
<body style="font-family:Arial,sans-serif;color:#333;max-width:600px;margin:0 auto;padding:20px">
  <h2 style="color:#1a73e8">New Contact Form Submission</h2>
  <table width="100%%" cellpadding="8" cellspacing="0" style="border-collapse:collapse">
    <tr><td style="background:#f5f5f5;width:160px"><strong>Submission ID</strong></td><td>%s</td></tr>
    <tr><td style="background:#f5f5f5"><strong>Date / Time</strong></td><td>%s</td></tr>
    <tr><td style="background:#f5f5f5"><strong>Full Name</strong></td><td>%s</td></tr>
    <tr><td style="background:#f5f5f5"><strong>Email</strong></td><td>%s</td></tr>
    <tr><td style="background:#f5f5f5"><strong>Company</strong></td><td>%s</td></tr>
    <tr><td style="background:#f5f5f5"><strong>Country</strong></td><td>%s</td></tr>
    <tr><td style="background:#f5f5f5"><strong>Phone</strong></td><td>%s</td></tr>
    <tr><td style="background:#f5f5f5"><strong>Subject</strong></td><td>%s</td></tr>
    <tr><td style="background:#f5f5f5;vertical-align:top"><strong>Message</strong></td><td style="white-space:pre-wrap">%s</td></tr>
    %s
  </table>
</body>
</html>`,
			submissionID,
			submittedAt,
			fullName,
			strVal(data["email"]),
			strVal(data["company"]),
			strVal(data["country"]),
			strVal(data["phone"]),
			strVal(data["subject"]),
			strVal(data["message"]),
			metaRows,
		)
	} else {
		subject = fmt.Sprintf("[Request Sample] New Submission – %s", fullName)
		body = fmt.Sprintf(`<!DOCTYPE html>
<html>
<body style="font-family:Arial,sans-serif;color:#333;max-width:600px;margin:0 auto;padding:20px">
  <h2 style="color:#1a73e8">New Request Sample Submission</h2>
  <table width="100%%" cellpadding="8" cellspacing="0" style="border-collapse:collapse">
    <tr><td style="background:#f5f5f5;width:160px"><strong>Submission ID</strong></td><td>%s</td></tr>
    <tr><td style="background:#f5f5f5"><strong>Date / Time</strong></td><td>%s</td></tr>
    <tr><td style="background:#f5f5f5"><strong>Full Name</strong></td><td>%s</td></tr>
    <tr><td style="background:#f5f5f5"><strong>Email</strong></td><td>%s</td></tr>
    <tr><td style="background:#f5f5f5"><strong>Company</strong></td><td>%s</td></tr>
    <tr><td style="background:#f5f5f5"><strong>Job Title</strong></td><td>%s</td></tr>
    <tr><td style="background:#f5f5f5"><strong>Country</strong></td><td>%s</td></tr>
    <tr><td style="background:#f5f5f5"><strong>Phone</strong></td><td>%s</td></tr>
    <tr><td style="background:#f5f5f5"><strong>Report Title</strong></td><td>%s</td></tr>
    <tr><td style="background:#f5f5f5;vertical-align:top"><strong>Additional Info</strong></td><td style="white-space:pre-wrap">%s</td></tr>
    %s
  </table>
</body>
</html>`,
			submissionID,
			submittedAt,
			fullName,
			strVal(data["email"]),
			strVal(data["company"]),
			strVal(data["jobTitle"]),
			strVal(data["country"]),
			strVal(data["phone"]),
			strVal(data["reportTitle"]),
			strVal(data["additionalInfo"]),
			metaRows,
		)
	}

	return subject, body
}

func buildMetadataRows(meta form.SubmissionMetadata) string {
	if meta.IPAddress == "" && meta.PageURL == "" && meta.Referrer == "" {
		return ""
	}
	rows := `<tr><td colspan="2" style="background:#e8eaf6;padding:8px"><strong>Submission Source</strong></td></tr>`
	if meta.IPAddress != "" {
		rows += fmt.Sprintf(`<tr><td style="background:#f0f4ff;width:160px"><strong>IP Address</strong></td><td>%s</td></tr>`, meta.IPAddress)
	}
	if meta.PageURL != "" {
		rows += fmt.Sprintf(`<tr><td style="background:#f0f4ff"><strong>Page URL</strong></td><td style="word-break:break-all">%s</td></tr>`, meta.PageURL)
	}
	if meta.Referrer != "" {
		rows += fmt.Sprintf(`<tr><td style="background:#f0f4ff"><strong>Referrer</strong></td><td style="word-break:break-all">%s</td></tr>`, meta.Referrer)
	}
	return rows
}

// SendOrderConfirmation sends an HTML order confirmation email to the customer.
func (s *smtpEmailService) SendOrderConfirmation(o *order.Order) error {
	subject := fmt.Sprintf("Your Order Confirmation — %s", o.ReportTitle)
	body := fmt.Sprintf(`<!DOCTYPE html>
<html>
<body style="font-family:Arial,sans-serif;color:#333;max-width:600px;margin:0 auto;padding:20px">
  <h2 style="color:#1a73e8">Order Confirmation</h2>
  <p>Dear %s,</p>
  <p>Thank you for your purchase! Your order has been received and is being processed.</p>
  <table width="100%%" cellpadding="8" cellspacing="0" style="border-collapse:collapse;margin:20px 0">
    <tr><td style="background:#f5f5f5;width:160px"><strong>Order ID</strong></td><td>#%d</td></tr>
    <tr><td style="background:#f5f5f5"><strong>Report</strong></td><td>%s</td></tr>
    <tr><td style="background:#f5f5f5"><strong>Amount Paid</strong></td><td>%s %.2f</td></tr>
    <tr><td style="background:#f5f5f5"><strong>Date</strong></td><td>%s</td></tr>
  </table>
  <p>Your report will be delivered to <strong>%s</strong> within <strong>2–3 business days</strong>.</p>
  <p>If you have any questions, please contact us at <a href="mailto:support@healthcareforesights.com">support@healthcareforesights.com</a>.</p>
  <p>Thank you for choosing HealthcareForesights!</p>
</body>
</html>`,
		o.CustomerName,
		o.ID,
		o.ReportTitle,
		o.Currency,
		o.Amount,
		o.CreatedAt.Format("January 2, 2006"),
		o.CustomerEmail,
	)

	m := gomail.NewMessage()
	m.SetHeader("From", s.cfg.From)
	m.SetHeader("To", o.CustomerEmail)
	m.SetHeader("Subject", subject)
	m.SetBody("text/html", body)

	d := gomail.NewDialer(s.cfg.Host, s.cfg.Port, s.cfg.User, s.cfg.Password)
	return d.DialAndSend(m)
}

// SendOrderAdminNotification sends an HTML notification email to the admin.
func (s *smtpEmailService) SendOrderAdminNotification(o *order.Order) error {
	subject := fmt.Sprintf("[NEW ORDER] Payment Received — %s — $%.2f", o.ReportTitle, o.Amount)
	body := fmt.Sprintf(`<!DOCTYPE html>
<html>
<body style="font-family:Arial,sans-serif;color:#333;max-width:600px;margin:0 auto;padding:20px">
  <h2 style="color:#e53935">New Order — Action Required</h2>
  <p>A new order has been placed and payment received. Please deliver the report PDF to the customer within 2–3 business days.</p>
  <h3 style="color:#1a73e8">Customer Details</h3>
  <table width="100%%" cellpadding="8" cellspacing="0" style="border-collapse:collapse;margin:0 0 20px 0">
    <tr><td style="background:#f5f5f5;width:160px"><strong>Name</strong></td><td>%s</td></tr>
    <tr><td style="background:#f5f5f5"><strong>Email</strong></td><td>%s</td></tr>
    <tr><td style="background:#f5f5f5"><strong>Company</strong></td><td>%s</td></tr>
    <tr><td style="background:#f5f5f5"><strong>Phone</strong></td><td>%s</td></tr>
    <tr><td style="background:#f5f5f5"><strong>Country</strong></td><td>%s</td></tr>
  </table>
  <h3 style="color:#1a73e8">Order Details</h3>
  <table width="100%%" cellpadding="8" cellspacing="0" style="border-collapse:collapse">
    <tr><td style="background:#f5f5f5;width:160px"><strong>Order ID</strong></td><td>#%d</td></tr>
    <tr><td style="background:#f5f5f5"><strong>Report</strong></td><td>%s</td></tr>
    <tr><td style="background:#f5f5f5"><strong>Amount</strong></td><td>%s %.2f</td></tr>
    <tr><td style="background:#f5f5f5"><strong>PayPal Order ID</strong></td><td>%s</td></tr>
    <tr><td style="background:#f5f5f5"><strong>PayPal Capture ID</strong></td><td>%s</td></tr>
    <tr><td style="background:#f5f5f5"><strong>Date</strong></td><td>%s</td></tr>
  </table>
  <p style="margin-top:20px">
    <a href="https://admin.healthcareforesights.com/orders/%d" style="background:#1a73e8;color:#fff;padding:10px 20px;text-decoration:none;border-radius:4px">View Order in Admin</a>
  </p>
</body>
</html>`,
		o.CustomerName,
		o.CustomerEmail,
		o.CustomerCompany,
		o.CustomerPhone,
		o.CustomerCountry,
		o.ID,
		o.ReportTitle,
		o.Currency,
		o.Amount,
		o.PaypalOrderID,
		o.PaypalCaptureID,
		o.CreatedAt.Format(time.RFC1123),
		o.ID,
	)

	m := gomail.NewMessage()
	m.SetHeader("From", s.cfg.From)
	m.SetHeader("To", s.cfg.NotifyTo)
	m.SetHeader("Subject", subject)
	m.SetBody("text/html", body)

	d := gomail.NewDialer(s.cfg.Host, s.cfg.Port, s.cfg.User, s.cfg.Password)
	return d.DialAndSend(m)
}

func strVal(v interface{}) string {
	if v == nil {
		return ""
	}
	s, _ := v.(string)
	return s
}
