package main

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"encoding/json"

	"lockcenter-backend/internal/config"
	"lockcenter-backend/internal/domain"
	"lockcenter-backend/internal/infrastructure/persistence"
	"lockcenter-backend/internal/infrastructure/security"

	"github.com/google/uuid"
	"github.com/xuri/excelize/v2"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil { log.Fatal(err) }
	db, err := persistence.NewDatabase(cfg)
	if err != nil { log.Fatal(err) }

	err = persistence.RunMigrations(db.Gorm)
	if err != nil { log.Fatal(err) }

	userRepo := persistence.NewGormUserRepository(db.Gorm)
	branchRepo := persistence.NewGormBranchRepository(db.Gorm)
	visitRepo := persistence.NewGormVisitRepository(db.Gorm)
	clientRepo := persistence.NewGormClientRepository(db.Gorm)
	hasher := security.NewBcryptHasher(12)
	ctx := context.Background()

	defaultBranchID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	branch, _ := branchRepo.GetByID(ctx, defaultBranchID)
	if branch == nil {
		branch = &domain.Branch{ID: defaultBranchID, Name: "Sede São Luís (Importado)", City: "São Luís", UF: "MA", Status: domain.BranchStatusActive}
		branchRepo.Create(ctx, branch)
	}

	db.Gorm.Exec("DELETE FROM visit_photos WHERE bucket_key = 'imported_link'")
	db.Gorm.Exec("DELETE FROM visits WHERE observations LIKE 'Importado da planilha%' OR receiver_name != ''")

	files, _ := filepath.Glob("../arquivos/*.xlsx")
	for _, file := range files {
		processFile(ctx, file, userRepo, visitRepo, clientRepo, hasher, defaultBranchID)
	}
	fmt.Println("\nImport task completed!")
}

type ColumnDefinition struct { Index int; OffsetFromEnd int }
type ColumnMap struct {
	Timestamp, SellerEmail, VisitDate, ClientName, CNPJ, ClientEmail, Receiver,
	ContactPhone, FixedPhone, Address, Subject, Conclusion, ArrivalTime, DepartureTime,
	KMStart, KMEnd, Photos ColumnDefinition
}

func processFile(ctx context.Context, filePath string, userRepo domain.UserRepository, visitRepo domain.VisitRepository, clientRepo domain.ClientRepository, hasher security.PasswordHasher, branchID uuid.UUID) {
	fmt.Printf("\n>>> Processing file: %s\n", filePath)
	f, err := excelize.OpenFile(filePath)
	if err != nil { fmt.Printf("Error: %v\n", err); return }
	defer f.Close()
	sheets := f.GetSheetList()
	if len(sheets) == 0 { return }
	rows, err := f.GetRows(sheets[0])
	if err != nil { fmt.Printf("Error rows: %v\n", err); return }
	if len(rows) < 2 { return }

	cmap := ColumnMap{}
	reset := func(c *ColumnDefinition) { c.Index = -1; c.OffsetFromEnd = -1 }
	reset(&cmap.Timestamp); reset(&cmap.SellerEmail); reset(&cmap.VisitDate); reset(&cmap.ClientName); reset(&cmap.CNPJ); reset(&cmap.ClientEmail); reset(&cmap.Receiver)
	reset(&cmap.ContactPhone); reset(&cmap.FixedPhone); reset(&cmap.Address); reset(&cmap.Subject); reset(&cmap.Conclusion); reset(&cmap.ArrivalTime); reset(&cmap.DepartureTime); reset(&cmap.KMStart); reset(&cmap.KMEnd); reset(&cmap.Photos)

	header := rows[0]

	for i, col := range header {
		colLower := strings.ToLower(strings.TrimSpace(col))
		
		switch {
		case strings.Contains(colLower, "registre as foto") || strings.Contains(colLower, "fotos"):
				if cmap.Photos.Index == -1 { cmap.Photos.Index = i }
		case strings.Contains(colLower, "e-mail do cliente") || strings.Contains(colLower, "email do cliente"):
				cmap.ClientEmail.Index = i
		case strings.Contains(colLower, "endereço de e-mail") || strings.Contains(colLower, "email do vendedor"):
				cmap.SellerEmail.Index = i
		case strings.Contains(colLower, "data da visita"):
				cmap.VisitDate.Index = i
		case strings.Contains(colLower, "quem lhe recebeu") || strings.Contains(colLower, "recebeu"):
				cmap.Receiver.Index = i
		case strings.Contains(colLower, "contato telefônico") || strings.Contains(colLower, "telefone celular"):
				cmap.ContactPhone.Index = i
		case strings.Contains(colLower, "telefone fixo") || strings.Contains(colLower, "fixo"):
				cmap.FixedPhone.Index = i
		case strings.Contains(colLower, "endereço do cliente"):
				cmap.Address.Index = i
		case (strings.Contains(colLower, "horário de chegada") || strings.Contains(colLower, "horario de chegada")) && !strings.Contains(colLower, "registre"): 
				cmap.ArrivalTime.Index = i 
		case (strings.Contains(colLower, "horário de saída") || strings.Contains(colLower, "horario de saida")) && !strings.Contains(colLower, "registre"): 
				cmap.DepartureTime.Index = i 
		case strings.Contains(colLower, "km inicial"):
				cmap.KMStart.Index = i
		case strings.Contains(colLower, "km de chegada") || strings.Contains(colLower, "km final"): 
				cmap.KMEnd.Index = i 
		case strings.Contains(colLower, "assunto"):
				cmap.Subject.Index = i
		case strings.Contains(colLower, "conclusão") || strings.Contains(colLower, "conclusao"):
				cmap.Conclusion.Index = i
		case strings.Contains(colLower, "cnpj"):
				cmap.CNPJ.Index = i
		case strings.Contains(colLower, "carimbo"):
				cmap.Timestamp.Index = i
		case strings.Contains(colLower, "cliente"):
				if cmap.ClientName.Index == -1 { cmap.ClientName.Index = i }
		}
	}

	fmt.Printf("Mapped Indices: Subject=%d, Conclusion=%d, Receiver=%d, Fixed=%d\n", 
		cmap.Subject.Index, cmap.Conclusion.Index, cmap.Receiver.Index, cmap.FixedPhone.Index)

	importedCount, photosCount := 0, 0
	for ri, row := range rows {
		if ri == 0 { continue }
		rl := len(row)
		get := func(c ColumnDefinition) string {
			if c.Index >= 0 && c.Index < rl { return strings.TrimSpace(row[c.Index]) }
			return ""
		}
		
		var email string
		if cmap.SellerEmail.Index != -1 {
			email = get(cmap.SellerEmail)
		}
		
		// Fallback for email if not found in specific column
		if email == "" || !strings.Contains(email, "@") {
			// Search for any email in the row
			for _, cell := range row {
				if strings.Contains(cell, "@") {
					email = strings.TrimSpace(cell)
					break
				}
			}
		}

		if email == "" || !strings.Contains(email, "@") { continue }
		
		user, _ := userRepo.GetByEmail(ctx, email)
		if user == nil {
			hash, _ := hasher.Hash("importado123")
			user = &domain.User{ID: uuid.New(), Name: strings.Split(email, "@")[0], Email: email, CPF: fmt.Sprintf("IMP-%s", uuid.New().String()[:8]), PasswordHash: hash, Role: domain.RoleSalesperson, Status: domain.UserStatusActive, MustChangePassword: true, BranchID: &branchID, CreatedAt: time.Now(), UpdatedAt: time.Now()}
			userRepo.Create(ctx, user)
		}

		cnpj := get(cmap.CNPJ)
		clientName := get(cmap.ClientName)

		if clientName == "" {
			clientName = "Cliente Não Informado"
		}
		
		if cnpj == "" {
			cnpj = fmt.Sprintf("IMP-%s", uuid.New().String()[:12])
		}

		addressMap := map[string]string{"street": get(cmap.Address)}
		addressBytes, _ := json.Marshal(addressMap)
		addressJSON := string(addressBytes)

		client, err := clientRepo.GetByCNPJ(ctx, cnpj, branchID)
		if err != nil || client == nil {
			client = &domain.Client{
				ID:           uuid.New(),
				BranchID:     branchID,
				SellerID:     user.ID,
				Name:         clientName,
				CNPJ:         cnpj,
				Email:        get(cmap.ClientEmail),
				ContactPhone: get(cmap.ContactPhone),
				FixedPhone:   get(cmap.FixedPhone),
				Address:      addressJSON,
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
			}
			clientRepo.Create(ctx, client)
		} else {
			client.Name = clientName
			client.SellerID = user.ID
			client.Email = get(cmap.ClientEmail)
			client.ContactPhone = get(cmap.ContactPhone)
			client.FixedPhone = get(cmap.FixedPhone)
			client.Address = addressJSON
			client.UpdatedAt = time.Now()
			clientRepo.Update(ctx, client)
		}

		rawSubject := strings.ToLower(strings.TrimSpace(get(cmap.Subject)))
		subject := string(domain.VisitSubjectProspeccao)
		switch {
		case strings.Contains(rawSubject, "prospec"): subject = string(domain.VisitSubjectProspeccao)
		case strings.Contains(rawSubject, "manuten"): subject = string(domain.VisitSubjectManutencao)
		case strings.Contains(rawSubject, "cobran"): subject = string(domain.VisitSubjectCobranca)
		case strings.Contains(rawSubject, "entrega"): subject = string(domain.VisitSubjectEntrega)
		case strings.Contains(rawSubject, "retirada"): subject = string(domain.VisitSubjectRetirada)
		case strings.Contains(rawSubject, "pós") || strings.Contains(rawSubject, "pos"): subject = string(domain.VisitSubjectPosVenda)
		}

		visit := &domain.Visit{
			ID:                 uuid.New(),
			SalespersonID:      user.ID,
			Status:             domain.StatusCompleted,
			Date:               parseXlsxDate(get(cmap.VisitDate)),
			ClientName:         clientName,
			ClientCNPJ:         cnpj,
			ClientEmail:        get(cmap.ClientEmail),
			ContactPhone:       get(cmap.ContactPhone),
			BranchPhone:        get(cmap.FixedPhone),
			Address:            get(cmap.Address),
			Subject:            subject,
			Conclusion:         get(cmap.Conclusion),
			ArrivalTime:        parseXlsxTime(get(cmap.VisitDate), get(cmap.ArrivalTime)),
			DepartureTime:      parseXlsxTime(get(cmap.VisitDate), get(cmap.DepartureTime)),
			KMStart:            parseKMSafe(get(cmap.KMStart)),
			KMEnd:              parseKMSafe(get(cmap.KMEnd)),
			ReceiverName:       get(cmap.Receiver),
			Observations:       "Importado da planilha comercial",
			CreatedAt:          time.Now(),
			UpdatedAt:          time.Now(),
		}

		if err := visitRepo.Create(ctx, visit); err == nil {
			if cmap.Photos.Index != -1 {
				for pi := cmap.Photos.Index; pi < rl; pi++ {
					u := strings.TrimSpace(row[pi])
					if u == "" { continue }
					links := strings.Fields(strings.ReplaceAll(u, ",", " "))
					for _, fu := range links {
						if !strings.Contains(fu, "http") { continue }
						p := &domain.VisitPhoto{
							ID:        uuid.New(),
							VisitID:   visit.ID,
							BucketKey: "imported_link",
							PublicURL: fu,
							FileName:  "drive_photo.link",
							PhotoType: "outros",
							CreatedAt: time.Now(),
						}
						visitRepo.AddPhoto(ctx, p)
						photosCount++
					}
				}
			}
			importedCount++
		}
	}
	fmt.Printf("Successfully imported %d visits and %d photos from %s\n", importedCount, photosCount, filePath)
}

func parseXlsxDate(v string) time.Time {
	v = strings.TrimSpace(v)
	if v == "" { return time.Now() }
	for _, f := range []string{"1/2/06", "01/02/2006", "2/1/06", "02/01/2006", "2006-01-02", "1/2/2006", "01-02-06", "02-01-06", "1/2/2006 15:04:05"} {
		if t, err := time.Parse(f, v); err == nil { return t }
	}
	return time.Now()
}

func parseXlsxTime(dp, tp string) *time.Time {
	tp = strings.TrimSpace(tp)
	if tp == "" { return nil }
	var pt time.Time
	var found bool
	formats := []string{
		"3:04:05 PM", "15:04:04", "15:04", "3:04 PM", 
		"15:04:00", "03:04:00 PM", "3:04:00 PM",
		"15:04:05.000", "3:04:05 PM",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, tp); err == nil {
			pt = t
			found = true
			break
		}
	}
	if !found {
		if f, err := strconv.ParseFloat(tp, 64); err == nil {
			seconds := f * 86400
			pt = time.Date(0, 1, 1, 0, 0, 0, 0, time.UTC).Add(time.Duration(seconds) * time.Second)
			found = true
		}
	}
	if !found { return nil }
	d := parseXlsxDate(dp)
	r := time.Date(d.Year(), d.Month(), d.Day(), pt.Hour(), pt.Minute(), pt.Second(), 0, time.UTC)
	return &r
}

func parseKMSafe(v string) *float64 {
	v = strings.TrimSpace(v)
	if v == "" { return nil }
	v = strings.ReplaceAll(v, ",", ".")
	if f, err := strconv.ParseFloat(v, 64); err == nil { return &f }
	return nil
}
