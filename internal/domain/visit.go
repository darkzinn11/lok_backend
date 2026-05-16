package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type VisitStatus string

const (
	StatusPending    VisitStatus = "Pendente"
	StatusInAnalysis VisitStatus = "Análise"
	StatusCompleted  VisitStatus = "Enviado"
	StatusCanceled   VisitStatus = "Cancelado"
	StatusDraft      VisitStatus = "Rascunho"
)

const (
	VisitSubjectProspeccao VisitSubject = "Prospecção"
	VisitSubjectManutencao VisitSubject = "Manutenção"
	VisitSubjectCobranca   VisitSubject = "Cobrança"
	VisitSubjectEntrega    VisitSubject = "Entrega"
	VisitSubjectRetirada   VisitSubject = "Retirada"
	VisitSubjectPosVenda   VisitSubject = "Pós-Venda"
)

type VisitSubject string

type Visit struct {
	ID                 uuid.UUID    `json:"id"`
	SalespersonID      uuid.UUID    `json:"salespersonId"`
	Status             VisitStatus  `json:"status"`
	Date               time.Time    `json:"date"`
	ClientName         string       `json:"clientName"`
	ClientCNPJ         string       `json:"clientCnpj"`
	ClientEmail        string       `json:"clientEmail"`
	ContactPhone       string       `json:"contactPhone"`
	BranchPhone        string       `json:"branchPhone"`
	Address            string       `json:"address"`
	Subject            string       `json:"subject"`
	Conclusion         string       `json:"conclusion"`
	ArrivalTime        *time.Time   `json:"arrivalTime"`
	DepartureTime      *time.Time   `json:"departureTime"`
	KMStart            *float64     `json:"kmStart"`
	KMEnd              *float64     `json:"kmEnd"`
	GeoLatitude        *float64     `json:"geoLatitude"`
	GeoLongitude       *float64     `json:"geoLongitude"`
	GeoAccuracyMeters  *float64     `json:"geoAccuracyMeters"`
	GeoCapturedAt      *time.Time   `json:"geoCapturedAt"`
	GeoReverseAddress  string       `json:"geoReverseAddress"`
	Photos             []VisitPhoto `json:"photos"`
	ReceiverName       string       `json:"receiverName"`
	Observations       string       `json:"observations"`
	ManagerObservation string       `json:"managerObservation"`
	CreatedAt          time.Time    `json:"createdAt"`
	UpdatedAt          time.Time    `json:"updatedAt"`
}

type VisitPhoto struct {
	ID        uuid.UUID `json:"id"`
	VisitID   uuid.UUID `json:"visitId"`
	BucketKey string    `json:"bucketKey"`
	PublicURL string    `json:"publicUrl"`
	FileName  string    `json:"fileName"`
	FileSize  int64     `json:"fileSize"`
	PhotoType string    `json:"photoType"`
	CreatedAt time.Time `json:"createdAt"`
}

type VisitFilters struct {
	SalespersonID *uuid.UUID
	BranchID      *uuid.UUID
	Search        string
	Status        *VisitStatus
	Subject       string
	Conclusion    string
	Date          *time.Time
	StartDate     *time.Time
	EndDate       *time.Time
	OnlyAlerts    bool
	Limit         int
	Offset        int
}

type VisitRepository interface {
	Create(ctx context.Context, visit *Visit) error
	GetByID(ctx context.Context, id uuid.UUID) (*Visit, error)
	List(ctx context.Context, filters VisitFilters) ([]*Visit, error)
	Count(ctx context.Context, filters VisitFilters) (int64, error)
	Update(ctx context.Context, visit *Visit) error
	UpdateStatus(ctx context.Context, id uuid.UUID, status VisitStatus) error
	AddPhoto(ctx context.Context, photo *VisitPhoto) error
	Delete(ctx context.Context, id uuid.UUID) error
}

func IsValidVisitSubject(value string) bool {
	switch VisitSubject(value) {
	case VisitSubjectProspeccao,
		VisitSubjectManutencao,
		VisitSubjectCobranca,
		VisitSubjectEntrega,
		VisitSubjectRetirada,
		VisitSubjectPosVenda:
		return true
	default:
		return false
	}
}
