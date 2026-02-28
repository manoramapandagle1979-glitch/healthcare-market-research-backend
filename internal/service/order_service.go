package service

import (
	"fmt"

	"github.com/healthcare-market-research/backend/internal/domain/order"
	"github.com/healthcare-market-research/backend/internal/repository"
	"github.com/healthcare-market-research/backend/pkg/email"
	"github.com/healthcare-market-research/backend/pkg/logger"
	"github.com/healthcare-market-research/backend/pkg/paypal"
)

type OrderService interface {
	CreateOrder(req *order.CreateOrderRequest) (*order.CreateOrderResponse, error)
	CaptureOrder(orderID uint) (*order.CaptureOrderResponse, error)
	GetAll(query order.GetOrdersQuery) ([]order.Order, int64, error)
	GetByID(id uint) (*order.Order, error)
	GetStats() (*order.OrderStats, error)
	UpdateStatus(id uint, req *order.UpdateStatusRequest, updatedBy *uint) error
}

type orderService struct {
	repo         repository.OrderRepository
	reportRepo   repository.ReportRepository
	paypalClient *paypal.Client
	emailSvc     email.EmailService
}

func NewOrderService(
	repo repository.OrderRepository,
	reportRepo repository.ReportRepository,
	paypalClient *paypal.Client,
	emailSvc email.EmailService,
) OrderService {
	return &orderService{
		repo:         repo,
		reportRepo:   reportRepo,
		paypalClient: paypalClient,
		emailSvc:     emailSvc,
	}
}

func (s *orderService) CreateOrder(req *order.CreateOrderRequest) (*order.CreateOrderResponse, error) {
	// Validate required fields
	if req.CustomerName == "" {
		return nil, fmt.Errorf("customer_name is required")
	}
	if req.CustomerEmail == "" {
		return nil, fmt.Errorf("customer_email is required")
	}
	if req.ReportSlug == "" {
		return nil, fmt.Errorf("report_slug is required")
	}

	// Look up the report from DB (price comes from DB only, never from browser)
	report, err := s.reportRepo.GetBySlug(req.ReportSlug)
	if err != nil {
		return nil, fmt.Errorf("report not found: %s", req.ReportSlug)
	}

	// Use discounted price if available, else regular price
	price := report.Price
	if report.DiscountedPrice > 0 {
		price = report.DiscountedPrice
	}

	if price <= 0 {
		return nil, fmt.Errorf("report does not have a valid price")
	}

	// Create PayPal order first
	amountStr := fmt.Sprintf("%.2f", price)
	description := fmt.Sprintf("Healthcare Market Research Report: %s", report.Title)

	paypalOrder, err := s.paypalClient.CreateOrder(amountStr, "USD", description, fmt.Sprintf("report-%d", report.ID))
	if err != nil {
		return nil, fmt.Errorf("failed to create PayPal order: %w", err)
	}

	// Save order to DB
	o := &order.Order{
		CustomerName:    req.CustomerName,
		CustomerEmail:   req.CustomerEmail,
		CustomerCompany: req.CustomerCompany,
		CustomerPhone:   req.CustomerPhone,
		CustomerCountry: req.CustomerCountry,
		ReportID:        uint(report.ID),
		ReportTitle:     report.Title,
		ReportSlug:      report.Slug,
		Amount:          price,
		Currency:        "USD",
		PaypalOrderID:   paypalOrder.ID,
		Status:          order.StatusPendingPayment,
	}

	if err := s.repo.Create(o); err != nil {
		return nil, fmt.Errorf("failed to save order: %w", err)
	}

	return &order.CreateOrderResponse{
		OrderID:       o.ID,
		PaypalOrderID: paypalOrder.ID,
	}, nil
}

func (s *orderService) CaptureOrder(orderID uint) (*order.CaptureOrderResponse, error) {
	o, err := s.repo.GetByID(orderID)
	if err != nil {
		return nil, fmt.Errorf("order not found")
	}

	if o.Status != order.StatusPendingPayment {
		return nil, fmt.Errorf("order is not in pending_payment status (current: %s)", o.Status)
	}

	if o.PaypalOrderID == "" {
		return nil, fmt.Errorf("order has no PayPal order ID")
	}

	// Capture the PayPal payment
	captureID, captureStatus, err := s.paypalClient.CaptureOrder(o.PaypalOrderID)
	if err != nil {
		return nil, fmt.Errorf("PayPal capture failed: %w", err)
	}

	// Update DB status
	if err := s.repo.UpdateStatus(o.ID, order.StatusPaymentReceived, captureID, "", nil); err != nil {
		logger.Error("Failed to update order status after capture", "order_id", o.ID, "error", err)
		return nil, fmt.Errorf("failed to update order status")
	}

	// Reload order for email
	o.PaypalCaptureID = captureID
	o.Status = order.StatusPaymentReceived

	// Fire-and-forget email notifications
	go func() {
		if err := s.emailSvc.SendOrderConfirmation(o); err != nil {
			logger.Error("Failed to send order confirmation email", "order_id", o.ID, "error", err)
		}
	}()
	go func() {
		if err := s.emailSvc.SendOrderAdminNotification(o); err != nil {
			logger.Error("Failed to send order admin notification email", "order_id", o.ID, "error", err)
		}
	}()

	logger.Info("Order captured", "order_id", o.ID, "capture_status", captureStatus)

	return &order.CaptureOrderResponse{
		OrderID:         o.ID,
		Status:          order.StatusPaymentReceived,
		PaypalCaptureID: captureID,
	}, nil
}

func (s *orderService) GetAll(query order.GetOrdersQuery) ([]order.Order, int64, error) {
	if query.Page < 1 {
		query.Page = 1
	}
	if query.Limit < 1 || query.Limit > 100 {
		query.Limit = 20
	}
	return s.repo.GetAll(query)
}

func (s *orderService) GetByID(id uint) (*order.Order, error) {
	return s.repo.GetByID(id)
}

func (s *orderService) GetStats() (*order.OrderStats, error) {
	return s.repo.GetStats()
}

func (s *orderService) UpdateStatus(id uint, req *order.UpdateStatusRequest, updatedBy *uint) error {
	// Validate status transition
	validStatuses := map[order.OrderStatus]bool{
		order.StatusPendingPayment:  true,
		order.StatusPaymentReceived: true,
		order.StatusProcessing:      true,
		order.StatusDelivered:       true,
		order.StatusCancelled:       true,
		order.StatusRefunded:        true,
	}

	if !validStatuses[req.Status] {
		return fmt.Errorf("invalid status: %s", req.Status)
	}

	return s.repo.UpdateStatus(id, req.Status, "", req.AdminNotes, updatedBy)
}
