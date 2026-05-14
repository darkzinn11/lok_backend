package main

import (
	"fmt"
	"log"

	"github.com/xuri/excelize/v2"
)

func main() {
	f, err := excelize.OpenFile("../arquivos/Relatório Comercial_ Gabriel Antonio (respostas).xlsx")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		log.Fatal("No sheets found")
	}

	rows, err := f.GetRows(sheets[0])
	if err != nil {
		log.Fatal(err)
	}

	if len(rows) > 0 {
		fmt.Println("Columns in the first sheet:")
		for i, col := range rows[0] {
			fmt.Printf("%d: %s\n", i, col)
		}
	} else {
		fmt.Println("No rows found")
	}
}
