package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omamx/profile-service/internal/repository"
)

type ComplianceChecker struct {
	pool          *pgxpool.Pool
	empRepo       *repository.EmployeeRepository
	checkInterval time.Duration
	stopChan      chan struct{}
	wg            sync.WaitGroup
}

func NewComplianceChecker(pool *pgxpool.Pool, empRepo *repository.EmployeeRepository, checkInterval time.Duration) *ComplianceChecker {
	return &ComplianceChecker{
		pool:          pool,
		empRepo:       empRepo,
		checkInterval: checkInterval,
		stopChan:      make(chan struct{}),
	}
}

func (c *ComplianceChecker) Start(ctx context.Context) {
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		// Run initial scan on startup
		_ = c.RunComplianceScan(ctx)

		ticker := time.NewTicker(c.checkInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-c.stopChan:
				return
			case <-ticker.C:
				_ = c.RunComplianceScan(ctx)
			}
		}
	}()
}

func (c *ComplianceChecker) Stop() {
	select {
	case <-c.stopChan:
	default:
		close(c.stopChan)
	}
	c.wg.Wait()
}

// RunComplianceScan inspects expiring certificates and generates warnings in the outbox.
func (c *ComplianceChecker) RunComplianceScan(ctx context.Context) error {
	// Threshold 1: Expiring in 30 days
	expiring30, err := c.empRepo.FindExpiringCertificates(ctx, 30)
	if err != nil {
		return fmt.Errorf("failed scanning 30-day expiring certificates: %w", err)
	}

	now := time.Now().UTC()
	for _, emp := range expiring30 {
		if emp.FoodSafetyCertExpiry == nil {
			continue
		}

		daysRemaining := int(emp.FoodSafetyCertExpiry.Sub(now).Hours() / 24)
		warningLevel := "WARNING_30_DAYS"
		if daysRemaining <= 7 {
			warningLevel = "CRITICAL_7_DAYS"
		}

		payload, _ := json.Marshal(map[string]any{
			"employee_id":             emp.ID.String(),
			"employee_code":           emp.EmployeeCode,
			"full_name":               emp.FullName,
			"department_id":           emp.DepartmentID,
			"position":                emp.Position,
			"food_safety_cert_no":     emp.FoodSafetyCertNo,
			"food_safety_cert_expiry": emp.FoodSafetyCertExpiry.Format("2006-01-02"),
			"days_remaining":          daysRemaining,
			"warning_level":           warningLevel,
			"detected_at":             now.Format(time.RFC3339),
		})

		// Write event to outbox table
		query := `
			INSERT INTO outbox_events (id, aggregate_type, aggregate_id, event_type, payload, topic)
			VALUES ($1, 'EmployeeProfile', $2, 'StaffComplianceWarningEvent', $3, 'profile.events.v1')
		`
		_, _ = c.pool.Exec(ctx, query, uuid.New(), emp.ID.String(), payload)
	}

	return nil
}
