package db

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"

	"github.com/libsql/libsql-shell-go/pkg/shell/enums"
	"github.com/olekukonko/tablewriter"
)

type Printer interface {
	print(statementResult StatementResult, outF io.Writer) error
}

type ExplainQueryPrinter struct{}

func (eqp ExplainQueryPrinter) print(statementResult StatementResult, outF io.Writer) error {
	data := [][]string{}

	tableData, err := appendData(statementResult, data, TABLE)
	if err != nil {
		return err
	}

	root, err := BuildQueryPlanTree(tableData)
	if err != nil {
		return err
	}
	println("QUERY PLAN")
	PrintQueryPlanTree(root, "")

	return nil
}

type TablePrinter struct {
	withoutHeader bool
}

func (t TablePrinter) print(statementResult StatementResult, outF io.Writer) error {
	data := [][]string{}
	table := createTable(outF)
	if !t.withoutHeader {
		table.SetHeader(statementResult.ColumnNames)
	}

	tableData, err := appendData(statementResult, data, TABLE)
	if err != nil {
		return err
	}

	table.AppendBulk(tableData)
	table.Render()

	return nil
}

type CSVPrinter struct {
	withoutHeader bool
}

func (c CSVPrinter) print(statementResult StatementResult, outF io.Writer) error {
	data := [][]string{}
	if !c.withoutHeader {
		data = append(data, statementResult.ColumnNames)
	}

	csvData, err := appendData(statementResult, data, CSV)
	if err != nil {
		return err
	}

	csvWriter := csv.NewWriter(outF)
	err = csvWriter.WriteAll(csvData)
	if err != nil {
		return err
	}

	return nil
}

type JSONPrinter struct{}

func (c JSONPrinter) print(statementResult StatementResult, outF io.Writer) error {
	var data []map[string]interface{}

	for row := range statementResult.RowCh {
		if row.Err != nil {
			return row.Err
		}
		rowData := make(map[string]interface{})
		formattedRow, err := FormatData(row.Row, JSON)
		if err != nil {
			return err
		}
		for i, v := range statementResult.ColumnNames {
			rowData[v] = formattedRow[i]
		}
		data = append(data, rowData)
	}

	json, err := json.Marshal(data)
	if err != nil {
		return err
	}
	if string(json) != "null" {
		fmt.Fprintln(outF, string(json))
	}
	return nil
}

func appendData(statementResult StatementResult, data [][]string, mode FormatType) ([][]string, error) {
	for row := range statementResult.RowCh {
		if row.Err != nil {
			return [][]string{}, row.Err
		}
		formattedRow, err := FormatData(row.Row, mode)
		if err != nil {
			return [][]string{}, err
		}
		data = append(data, formattedRow)
	}

	return data, nil
}

func getPrinter(mode enums.PrintMode, withoutHeader bool, isExplainQueryPlan bool) (Printer, error) {
	if isExplainQueryPlan {
		return &ExplainQueryPrinter{}, nil
	}
	switch mode {
	case enums.TABLE_MODE:
		return &TablePrinter{
			withoutHeader: withoutHeader,
		}, nil
	case enums.CSV_MODE:
		return &CSVPrinter{
			withoutHeader: withoutHeader,
		}, nil
	case enums.JSON_MODE:
		return &JSONPrinter{}, nil
	default:
		return nil, fmt.Errorf("unsupported printer: %s", mode)
	}
}

func PrintStatementsResult(statementsResult StatementsResult, outF io.Writer, withoutHeader bool, mode enums.PrintMode) error {
	if statementsResult.StatementResultCh == nil {
		return &InvalidStatementsResult{}
	}

	for statementResult := range statementsResult.StatementResultCh {
		if statementResult.Err != nil {
			return statementResult.Err
		}

		err := PrintStatementResult(statementResult, outF, withoutHeader, mode)
		if err != nil {
			return err
		}
	}
	return nil
}

func PrintStatementResult(statementResult StatementResult, outF io.Writer, withoutHeader bool, mode enums.PrintMode) error {
	if statementResult.RowCh == nil {
		return &UnableToPrintStatementResult{}
	}

	isExplainQueryPlan := IsResultComingFromExplainQueryPlan(statementResult)
	printer, err := getPrinter(mode, withoutHeader, isExplainQueryPlan)
	if err != nil {
		return err
	}

	err = printer.print(statementResult, outF)
	if err != nil {
		return err
	}

	return nil
}

func PrintError(err error, errF io.Writer) {
	fmt.Fprintf(errF, "Error: %s\n", err.Error())
}

func createTable(outF io.Writer) *tablewriter.Table {
	table := tablewriter.NewWriter(outF)

	table.SetHeaderLine(false)
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetAutoFormatHeaders(true)

	table.SetBorder(false)
	table.SetAutoWrapText(false)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetColumnSeparator("  ")
	table.SetNoWhiteSpace(true)
	table.SetTablePadding("     ")

	return table
}

func PrintTable(outF io.Writer, header []string, data [][]string) {
	table := createTable(outF)
	table.SetHeader(header)
	table.AppendBulk(data)
	table.Render()
}
