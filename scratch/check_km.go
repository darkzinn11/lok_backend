package main
import (
	"fmt"
	"github.com/xuri/excelize/v2"
)
func main() {
	f, err := excelize.OpenFile("../arquivos/Relatório Comercial_ Gabriel Antonio (respostas).xlsx")
	if err != nil { fmt.Println(err); return }
	sheets := f.GetSheetList()
	fmt.Printf("Sheets: %v\n", sheets)
	if len(sheets) == 0 { return }
	rows, err := f.GetRows(sheets[0])
	if err != nil { fmt.Println(err); return }
	for i, row := range rows {
		if len(row) > 15 {
			fmt.Printf("Row %d: [%s] | [%s] (Client: %s)\n", i, row[14], row[15], row[3])
		}
	}
}
