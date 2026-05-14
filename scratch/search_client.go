package main
import (
	"fmt"
	"path/filepath"
	"strings"
	"github.com/xuri/excelize/v2"
)
func main() {
	files, _ := filepath.Glob("../arquivos/*.xlsx")
	for _, fn := range files {
		f, err := excelize.OpenFile(fn)
		if err != nil { continue }
		sheets := f.GetSheetList()
		if len(sheets) == 0 { continue }
		rows, err := f.GetRows(sheets[0])
		if err != nil { continue }
		for ri, row := range rows {
			for ci, col := range row {
				if strings.Contains(strings.ToLower(col), "fertgrow") {
					fmt.Printf("File: %s, Row %d, Col %d: %v\n", fn, ri, ci, row)
				}
			}
		}
		f.Close()
	}
}
