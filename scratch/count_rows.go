package main
import (
	"fmt"
	"github.com/xuri/excelize/v2"
	"path/filepath"
)
func main() {
	files, _ := filepath.Glob("../arquivos/*.xlsx")
	total := 0
	for _, fn := range files {
		f, _ := excelize.OpenFile(fn)
		sheets := f.GetSheetList()
		if len(sheets) == 0 { continue }
		rows, _ := f.GetRows(sheets[0])
		fmt.Printf("%s: %d rows\n", fn, len(rows))
		total += len(rows)
	}
	fmt.Printf("Total rows expected (approx, minus headers): %d\n", total)
}
