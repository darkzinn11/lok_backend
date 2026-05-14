package main
import (
	"fmt"
	"github.com/xuri/excelize/v2"
)
func main() {
	f, _ := excelize.OpenFile("../arquivos/Relatório Comercial_ Gabriel Ferreira (respostas) 2.xlsx")
	rows, _ := f.GetRows(f.GetSheetList()[0])
	for i, row := range rows {
		if len(row) > 13 {
			fmt.Printf("Row %d: [%s] | [%s]\n", i, row[12], row[13])
		}
	}
}
