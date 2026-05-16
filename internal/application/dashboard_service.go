package application

import (
	"context"
	"sort"
	"strings"
	"time"

	"lockcenter-backend/internal/domain"

	"github.com/google/uuid"
)

const criticalAlertThresholdDays = 90

type DashboardTrendPoint struct {
	Date  string `json:"date"`
	Count int    `json:"count"`
}

type DashboardSubjectItem struct {
	Name  string `json:"name"`
	Value int    `json:"value"`
}

type DashboardSellerPerformanceItem struct {
	SellerID string `json:"sellerId"`
	Name     string `json:"name"`
	Visits   int    `json:"visits"`
}

type DashboardOverviewOutput struct {
	TotalVisits           int                              `json:"totalVisits"`
	PendingVisits         int                              `json:"pendingVisits"`
	CompletedVisits       int                              `json:"completedVisits"`
	CriticalAlerts        int                              `json:"criticalAlerts"`
	VisitTrend            []DashboardTrendPoint            `json:"visitTrend"`
	SubjectDistribution   []DashboardSubjectItem           `json:"subjectDistribution"`
	SellerPerformance     []DashboardSellerPerformanceItem `json:"sellerPerformance"`
	SellerPerformanceYear []DashboardSellerPerformanceItem `json:"sellerPerformanceYear"`
	RecentVisits          []VisitOutput                    `json:"recentVisits"`
}

type SellerReportItem struct {
	SellerID        string     `json:"sellerId"`
	SellerName      string     `json:"sellerName"`
	BranchName      string     `json:"branchName"`
	TotalVisits     int        `json:"totalVisits"`
	CompletedVisits int        `json:"completedVisits"`
	PendingVisits   int        `json:"pendingVisits"`
	CompletionRate  float64    `json:"completionRate"`
	UniqueClients   int        `json:"uniqueClients"`
	LastVisitDate   *time.Time `json:"lastVisitDate"`
}

type SellerReportOutput struct {
	Items []SellerReportItem `json:"items"`
}

type DashboardRange struct {
	StartDate time.Time
	EndDate   time.Time
}

type DashboardService struct {
	visitRepo  domain.VisitRepository
	userRepo   domain.UserRepository
	sellerRepo domain.SellerRepository
	branchRepo domain.BranchRepository
	clientRepo domain.ClientRepository
}

func NewDashboardService(
	visitRepo domain.VisitRepository,
	userRepo domain.UserRepository,
	sellerRepo domain.SellerRepository,
	branchRepo domain.BranchRepository,
	clientRepo domain.ClientRepository,
) *DashboardService {
	return &DashboardService{
		visitRepo:  visitRepo,
		userRepo:   userRepo,
		sellerRepo: sellerRepo,
		branchRepo: branchRepo,
		clientRepo: clientRepo,
	}
}

func (s *DashboardService) GetOverview(ctx context.Context, actorID uuid.UUID, input DashboardRange) (*DashboardOverviewOutput, error) {
	actor, err := s.userRepo.GetByID(ctx, actorID)
	if err != nil {
		return nil, domain.ErrUnauthorized
	}

	if !actor.CanManageSellers() && !actor.IsSalesperson() {
		return nil, domain.ErrForbidden
	}

	startDate := normalizeDateBoundary(input.StartDate)
	endDate := normalizeDateBoundary(input.EndDate)
	if endDate.Before(startDate) {
		return nil, domain.ErrValidation
	}

	// Filter setup for the period
	baseFilters := domain.VisitFilters{
		StartDate: &startDate,
		EndDate:   &endDate,
	}
	if actor.IsSalesperson() {
		baseFilters.SalespersonID = &actor.ID
	} else if actor.IsManager() {
		baseFilters.BranchID = actor.BranchID
	}

	// 1. Total Visits (non-draft)
	totalVisits, err := s.visitRepo.Count(ctx, baseFilters)
	if err != nil {
		return nil, err
	}

	// 2. Pending Visits
	pendingFilters := baseFilters
	pendingStatus := domain.StatusPending
	pendingFilters.Status = &pendingStatus
	pendingVisits, err := s.visitRepo.Count(ctx, pendingFilters)
	if err != nil {
		return nil, err
	}

	// 3. Completed Visits
	completedFilters := baseFilters
	completedStatus := domain.StatusCompleted
	completedFilters.Status = &completedStatus
	completedVisits, err := s.visitRepo.Count(ctx, completedFilters)
	if err != nil {
		return nil, err
	}

	// 4. Critical Alerts (History-wide, not just period)
	alertFilters := domain.VisitFilters{OnlyAlerts: true}
	if actor.IsSalesperson() {
		alertFilters.SalespersonID = &actor.ID
	} else if actor.IsManager() {
		alertFilters.BranchID = actor.BranchID
	}
	criticalAlerts, err := s.visitRepo.Count(ctx, alertFilters)
	if err != nil {
		return nil, err
	}

	// 5. Recent Visits (Limit to 5)
	recentFilters := domain.VisitFilters{Limit: 5}
	if actor.IsSalesperson() {
		recentFilters.SalespersonID = &actor.ID
	} else if actor.IsManager() {
		recentFilters.BranchID = actor.BranchID
	}
	recentVisitsList, err := s.visitRepo.List(ctx, recentFilters)
	if err != nil {
		return nil, err
	}

	visitService := NewVisitService(s.visitRepo, s.userRepo, s.branchRepo, s.clientRepo)
	recentVisits, err := visitService.mapVisits(ctx, recentVisitsList)
	if err != nil {
		return nil, err
	}

	// For charts and seller performance, we still need some data aggregation.
	// In a high-traffic system, we would use specialized GROUP BY queries.
	// For now, we will fetch only the essential fields or keep the current in-memory 
	// for the SMALL set of visits in the period, which is much better than ALL history.
	periodVisits, err := s.visitRepo.List(ctx, baseFilters)
	if err != nil {
		return nil, err
	}

	scopedSellers, err := s.listScopedSellers(ctx, actor)
	if err != nil {
		return nil, err
	}

	return &DashboardOverviewOutput{
		TotalVisits:           int(totalVisits),
		PendingVisits:         int(pendingVisits),
		CompletedVisits:       int(completedVisits),
		CriticalAlerts:        int(criticalAlerts),
		VisitTrend:            buildVisitTrend(periodVisits, startDate, endDate),
		SubjectDistribution:   buildSubjectDistribution(periodVisits),
		SellerPerformance:     buildSellerPerformance(scopedSellers, periodVisits, startDate, endDate),
		SellerPerformanceYear: buildSellerPerformanceYear(scopedSellers, periodVisits, time.Now().UTC()),
		RecentVisits:          recentVisits,
	}, nil
}

func (s *DashboardService) GetSellerReport(ctx context.Context, actorID uuid.UUID, input DashboardRange) (*SellerReportOutput, error) {
	actor, err := s.userRepo.GetByID(ctx, actorID)
	if err != nil {
		return nil, domain.ErrUnauthorized
	}

	if !actor.CanManageSellers() {
		return nil, domain.ErrForbidden
	}

	startDate := normalizeDateBoundary(input.StartDate)
	endDate := normalizeDateBoundary(input.EndDate)
	if endDate.Before(startDate) {
		return nil, domain.ErrValidation
	}

	scopedVisits, err := s.listScopedVisits(ctx, actor)
	if err != nil {
		return nil, err
	}

	scopedSellers, err := s.listScopedSellers(ctx, actor)
	if err != nil {
		return nil, err
	}

	// Fetch all branches to get names
	branches, err := s.branchRepo.List(ctx)
	if err != nil {
		return nil, err
	}
	branchMap := make(map[uuid.UUID]string)
	for _, b := range branches {
		branchMap[b.ID] = b.Name
	}

	periodVisits := filterVisitsByDate(scopedVisits, startDate, endDate)

	// Group visits by seller
	visitsBySeller := make(map[uuid.UUID][]*domain.Visit)
	for _, v := range periodVisits {
		visitsBySeller[v.SalespersonID] = append(visitsBySeller[v.SalespersonID], v)
	}

	// For LastVisitDate, we look at ALL visits, not just periodVisits
	allVisitsBySeller := make(map[uuid.UUID][]*domain.Visit)
	for _, v := range scopedVisits {
		allVisitsBySeller[v.SalespersonID] = append(allVisitsBySeller[v.SalespersonID], v)
	}

	items := make([]SellerReportItem, 0, len(scopedSellers))
	for _, seller := range scopedSellers {
		sVisits := visitsBySeller[seller.ID]
		allSVisits := allVisitsBySeller[seller.ID]

		completed := 0
		pending := 0
		uniqueClients := make(map[string]bool)

		for _, v := range sVisits {
			if v.Status == domain.StatusDraft {
				continue
			}

			if v.Status == domain.StatusCompleted {
				completed++
			} else if v.Status == domain.StatusPending || v.Status == domain.StatusInAnalysis {
				pending++
			}

			// Use CNPJ or Name as client identifier
			clientKey := v.ClientCNPJ
			if clientKey == "" {
				clientKey = v.ClientName
			}
			if clientKey != "" {
				uniqueClients[clientKey] = true
			}
		}

		total := completed + pending
		rate := 0.0
		if total > 0 {
			rate = (float64(completed) / float64(total)) * 100
		}

		var lastVisit *time.Time
		if len(allSVisits) > 0 {
			latest := allSVisits[0].Date
			for _, v := range allSVisits {
				if v.Date.After(latest) {
					latest = v.Date
				}
			}
			lastVisit = &latest
		}

		branchName := ""
		if seller.BranchID != nil {
			branchName = branchMap[*seller.BranchID]
		}

		items = append(items, SellerReportItem{
			SellerID:        seller.ID.String(),
			SellerName:      seller.Name,
			BranchName:      branchName,
			TotalVisits:     total,
			CompletedVisits: completed,
			PendingVisits:   pending,
			CompletionRate:  rate,
			UniqueClients:   len(uniqueClients),
			LastVisitDate:   lastVisit,
		})
	}

	return &SellerReportOutput{Items: items}, nil
}

func (s *DashboardService) listScopedVisits(ctx context.Context, actor *domain.User) ([]*domain.Visit, error) {
	filters := domain.VisitFilters{}
	if actor.IsSalesperson() {
		filters.SalespersonID = &actor.ID
	} else if actor.IsManager() {
		if actor.BranchID == nil {
			return nil, domain.ErrForbidden
		}
		filters.BranchID = actor.BranchID
	}

	return s.visitRepo.List(ctx, filters)
}

func (s *DashboardService) listScopedSellers(ctx context.Context, actor *domain.User) ([]*domain.User, error) {
	if actor.IsSalesperson() {
		return []*domain.User{actor}, nil
	}

	filters := domain.SellerFilters{}
	if actor.IsManager() {
		if actor.BranchID == nil {
			return nil, domain.ErrForbidden
		}
		filters.BranchID = actor.BranchID
	}

	return s.sellerRepo.List(ctx, filters)
}

func (s *DashboardService) mapRecentVisits(ctx context.Context, visits []*domain.Visit) ([]VisitOutput, error) {
	if len(visits) == 0 {
		return []VisitOutput{}, nil
	}

	cloned := make([]*domain.Visit, len(visits))
	copy(cloned, visits)
	sort.Slice(cloned, func(i, j int) bool {
		if cloned[i].Date.Equal(cloned[j].Date) {
			return cloned[i].CreatedAt.After(cloned[j].CreatedAt)
		}
		return cloned[i].Date.After(cloned[j].Date)
	})

	if len(cloned) > 5 {
		cloned = cloned[:5]
	}

	visitService := NewVisitService(s.visitRepo, s.userRepo, s.branchRepo, s.clientRepo)
	return visitService.mapVisits(ctx, cloned)
}

func normalizeDateBoundary(value time.Time) time.Time {
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, time.UTC)
}

func filterVisitsByDate(visits []*domain.Visit, startDate, endDate time.Time) []*domain.Visit {
	filtered := make([]*domain.Visit, 0, len(visits))
	for _, visit := range visits {
		visitDate := normalizeDateBoundary(visit.Date)
		if (visitDate.Equal(startDate) || visitDate.After(startDate)) && (visitDate.Equal(endDate) || visitDate.Before(endDate)) {
			filtered = append(filtered, visit)
		}
	}
	return filtered
}

func countVisitsByStatus(visits []*domain.Visit, status domain.VisitStatus) int {
	count := 0
	for _, visit := range visits {
		if visit.Status == status {
			count++
		}
	}
	return count
}

func countCriticalAlerts(visits []*domain.Visit, now time.Time) int {
	count := 0
	cutoff := now.AddDate(0, 0, -criticalAlertThresholdDays)
	for _, visit := range visits {
		if visit.Status == domain.StatusCompleted || visit.Status == domain.StatusDraft {
			continue
		}
		if !visit.UpdatedAt.After(cutoff) {
			count++
		}
	}
	return count
}

func buildVisitTrend(visits []*domain.Visit, startDate, endDate time.Time) []DashboardTrendPoint {
	trendEnd := normalizeDateBoundary(endDate)
	trendStart := normalizeDateBoundary(startDate)

	daysDiff := int(trendEnd.Sub(trendStart).Hours() / 24)
	isMonthly := daysDiff > 90

	counts := make(map[string]int)
	for _, visit := range visits {
		visitDate := normalizeDateBoundary(visit.Date)
		if visitDate.Before(trendStart) || visitDate.After(trendEnd) {
			continue
		}
		
		var key string
		if isMonthly {
			key = visitDate.Format("2006-01")
		} else {
			key = visitDate.Format("2006-01-02")
		}
		counts[key]++
	}

	points := make([]DashboardTrendPoint, 0)
	
	if isMonthly {
		startM := time.Date(trendStart.Year(), trendStart.Month(), 1, 0, 0, 0, 0, time.UTC)
		endM := time.Date(trendEnd.Year(), trendEnd.Month(), 1, 0, 0, 0, 0, time.UTC)
		for current := startM; !current.After(endM); current = current.AddDate(0, 1, 0) {
			key := current.Format("2006-01")
			points = append(points, DashboardTrendPoint{
				Date:  current.Format("Jan/06"),
				Count: counts[key],
			})
		}
	} else {
		for current := trendStart; !current.After(trendEnd); current = current.AddDate(0, 0, 1) {
			key := current.Format("2006-01-02")
			points = append(points, DashboardTrendPoint{
				Date:  current.Format("02/01"),
				Count: counts[key],
			})
		}
	}

	return points
}

func buildSubjectDistribution(visits []*domain.Visit) []DashboardSubjectItem {
	if len(visits) == 0 {
		return []DashboardSubjectItem{}
	}

	grouped := make(map[string]int)
	for _, visit := range visits {
		raw := strings.ToLower(visit.Subject)
		cat := "Outros"
		switch {
		case strings.Contains(raw, "prospec"):
			cat = string(domain.VisitSubjectProspeccao)
		case strings.Contains(raw, "manuten"):
			cat = string(domain.VisitSubjectManutencao)
		case strings.Contains(raw, "cobra"):
			cat = string(domain.VisitSubjectCobranca)
		case strings.Contains(raw, "entrega"):
			cat = string(domain.VisitSubjectEntrega)
		case strings.Contains(raw, "retira"):
			cat = string(domain.VisitSubjectRetirada)
		case strings.Contains(raw, "pós"):
			cat = string(domain.VisitSubjectPosVenda)
		}
		grouped[cat]++
	}

	items := make([]DashboardSubjectItem, 0, len(grouped))
	for subject, count := range grouped {
		items = append(items, DashboardSubjectItem{Name: subject, Value: count})
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].Value == items[j].Value {
			return items[i].Name < items[j].Name
		}
		return items[i].Value > items[j].Value
	})

	return items
}

func buildSellerPerformance(sellers []*domain.User, visits []*domain.Visit, startDate, endDate time.Time) []DashboardSellerPerformanceItem {
	visitCountBySeller := make(map[uuid.UUID]int, len(sellers))
	for _, visit := range visits {
		visitDate := normalizeDateBoundary(visit.Date)
		if visitDate.Before(startDate) || visitDate.After(endDate) || visit.Status == domain.StatusCanceled || visit.Status == domain.StatusDraft {
			continue
		}
		visitCountBySeller[visit.SalespersonID]++
	}

	items := make([]DashboardSellerPerformanceItem, 0, len(sellers))
	for _, seller := range sellers {
		items = append(items, DashboardSellerPerformanceItem{
			SellerID: seller.ID.String(),
			Name:     seller.Name,
			Visits:   visitCountBySeller[seller.ID],
		})
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].Visits == items[j].Visits {
			return items[i].Name < items[j].Name
		}
		return items[i].Visits > items[j].Visits
	})

	return items
}

func buildSellerPerformanceYear(sellers []*domain.User, visits []*domain.Visit, now time.Time) []DashboardSellerPerformanceItem {
	startOfYear := time.Date(now.Year(), time.January, 1, 0, 0, 0, 0, time.UTC)

	visitCountBySeller := make(map[uuid.UUID]int, len(sellers))
	for _, visit := range visits {
		visitDate := normalizeDateBoundary(visit.Date)
		if visitDate.Before(startOfYear) || visit.Status == domain.StatusCanceled || visit.Status == domain.StatusDraft {
			continue
		}
		visitCountBySeller[visit.SalespersonID]++
	}

	items := make([]DashboardSellerPerformanceItem, 0, len(sellers))
	for _, seller := range sellers {
		items = append(items, DashboardSellerPerformanceItem{
			SellerID: seller.ID.String(),
			Name:     seller.Name,
			Visits:   visitCountBySeller[seller.ID],
		})
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].Visits == items[j].Visits {
			return items[i].Name < items[j].Name
		}
		return items[i].Visits > items[j].Visits
	})

	return items
}
func countNonDraftVisits(visits []*domain.Visit) int {
	count := 0
	for _, v := range visits {
		if v.Status != domain.StatusDraft {
			count++
		}
	}
	return count
}
