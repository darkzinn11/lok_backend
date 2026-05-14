package main
import (
	"fmt"
	"github.com/xuri/excelize/v2"
)
func main() {
	f, _ := excelize.OpenFile("../arquivos/Relatório Comercial_ Placido Cirt (respostas).xlsx")
	rows, _ := f.GetRows(f.GetSheetList()[0])
	for i := 0; i < 20; i++ {
		if i < len(rows) && len(rows[i]) > 15 {
			fmt.Printf("Row %d: [%s] | [%s]\n", i, rows[i][14], rows[i][15])
		}
	}
}
