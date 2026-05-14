package main
import (
	"fmt"
	"github.com/xuri/excelize/v2"
)
func main() {
	f, _ := excelize.OpenFile("../arquivos/Relatório Comercial_ Gabriel Ferreira (respostas) 2.xlsx")
	rows, _ := f.GetRows(f.GetSheetList()[0])
	for i, col := range rows[0] {
		fmt.Printf("%d: %s\n", i, col)
	}
}
