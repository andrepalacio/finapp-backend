package handlers

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/andrespalacio/finapp-backend/internal/middleware"
	"github.com/andrespalacio/finapp-backend/internal/services"
	"github.com/andrespalacio/finapp-backend/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/xuri/excelize/v2"
)

type importTransactionService interface {
	Create(ctx context.Context, p services.CreateTransactionParams) (services.TransactionView, error)
}

type importCategoryService interface {
	ListForWorkspace(ctx context.Context, workspaceID uuid.UUID) ([]services.CategoryView, error)
}

type ImportHandler struct {
	txSvc  importTransactionService
	catSvc importCategoryService
}

func NewImportHandler(txSvc importTransactionService, catSvc importCategoryService) *ImportHandler {
	return &ImportHandler{txSvc: txSvc, catSvc: catSvc}
}

var templateHeaders = []string{
	"fecha",
	"descripcion",
	"monto",
	"tipo",
	"categoria",
	"notas",
}

var templateExample = []string{
	"2024-01-15",
	"Mercado",
	"50000",
	"expense",
	"Alimentacion",
	"compra semanal",
}

// @Summary     Download import template
// @Tags        transactions
// @Produce     application/vnd.openxmlformats-officedocument.spreadsheetml.sheet
// @Security    BearerAuth
// @Param       workspace_id path string true "Workspace ID"
// @Success     200
// @Router      /workspaces/{workspace_id}/transactions/import/template [get]
func (h *ImportHandler) Template(c *gin.Context) {
	f := excelize.NewFile()
	defer f.Close() //nolint:errcheck

	sheet := "Transacciones"
	f.SetSheetName("Sheet1", sheet)

	// Headers — bold + background
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"D9E1F2"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center"},
	})

	for col, header := range templateHeaders {
		cell, _ := excelize.CoordinatesToCellName(col+1, 1)
		f.SetCellValue(sheet, cell, header)
		f.SetCellStyle(sheet, cell, cell, headerStyle)
	}

	// Example row
	for col, val := range templateExample {
		cell, _ := excelize.CoordinatesToCellName(col+1, 2)
		f.SetCellValue(sheet, cell, val)
	}

	// Column widths
	widths := []float64{14, 30, 12, 10, 20, 30}
	for i, w := range widths {
		col, _ := excelize.ColumnNumberToName(i + 1)
		f.SetColWidth(sheet, col, col, w) //nolint:errcheck
	}

	// Instructions sheet
	info := "Instrucciones"
	f.NewSheet(info)
	f.SetCellValue(info, "A1", "Campo")
	f.SetCellValue(info, "B1", "Requerido")
	f.SetCellValue(info, "C1", "Valores validos")
	rows := [][]string{
		{"fecha", "Si", "YYYY-MM-DD"},
		{"descripcion", "No", "Texto libre"},
		{"monto", "Si", "Numero positivo"},
		{"tipo", "Si", "expense, income"},
		{"categoria", "No", "Nombre exacto de categoria (dejar vacio si no aplica)"},
		{"notas", "No", "Texto libre"},
	}
	for i, row := range rows {
		for j, val := range row {
			cell, _ := excelize.CoordinatesToCellName(j+1, i+2)
			f.SetCellValue(info, cell, val)
		}
	}

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		response.HandleError(c, err)
		return
	}

	c.Header("Content-Disposition", "attachment; filename=plantilla_transacciones.xlsx")
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", buf.Bytes())
}

type importRowResult struct {
	Row     int     `json:"row"`
	Valid   bool    `json:"valid"`
	Error   string  `json:"error,omitempty"`
	Date    string  `json:"date,omitempty"`
	Desc    string  `json:"description,omitempty"`
	Amount  float64 `json:"amount,omitempty"`
	Type    string  `json:"type,omitempty"`
	Categ   string  `json:"category,omitempty"`
}

type importSummary struct {
	Total   int               `json:"total"`
	Imported int              `json:"imported"`
	Skipped  int              `json:"skipped"`
	Rows     []importRowResult `json:"rows"`
}

// @Summary     Import transactions from XLSX
// @Tags        transactions
// @Accept      multipart/form-data
// @Produce     json
// @Security    BearerAuth
// @Param       workspace_id path string true "Workspace ID"
// @Param       file formData file true "XLSX file"
// @Param       dry_run query boolean false "Preview only, do not save"
// @Success     200 {object} importSummary
// @Router      /workspaces/{workspace_id}/transactions/import [post]
func (h *ImportHandler) Import(c *gin.Context) {
	dryRun := c.Query("dry_run") == "true"

	file, err := c.FormFile("file")
	if err != nil {
		response.BadRequest(c, "INVALID_INPUT", "file is required")
		return
	}

	f, err := file.Open()
	if err != nil {
		response.BadRequest(c, "INVALID_INPUT", "cannot open file")
		return
	}
	defer f.Close()

	xlsx, err := excelize.OpenReader(f)
	if err != nil {
		response.BadRequest(c, "INVALID_INPUT", "invalid XLSX file")
		return
	}
	defer xlsx.Close() //nolint:errcheck

	sheets := xlsx.GetSheetList()
	if len(sheets) == 0 {
		response.BadRequest(c, "INVALID_INPUT", "empty workbook")
		return
	}

	// Use first sheet
	rows, err := xlsx.GetRows(sheets[0])
	if err != nil {
		response.BadRequest(c, "INVALID_INPUT", "cannot read sheet")
		return
	}

	if len(rows) < 2 {
		response.OK(c, importSummary{Rows: []importRowResult{}})
		return
	}

	wsID := middleware.WorkspaceIDFromContext(c)
	userID := middleware.UserIDFromContext(c)

	// Build category name → ID map
	catMap := map[string]uuid.UUID{}
	if cats, err := h.catSvc.ListForWorkspace(c.Request.Context(), wsID); err == nil {
		for _, cat := range cats {
			catMap[strings.ToLower(cat.Name)] = cat.ID
		}
	}

	results := make([]importRowResult, 0, len(rows)-1)
	imported := 0

	for rowIdx, cols := range rows[1:] {
		rowNum := rowIdx + 2
		res := importRowResult{Row: rowNum}

		get := func(i int) string {
			if i < len(cols) {
				return strings.TrimSpace(cols[i])
			}
			return ""
		}

		dateStr := get(0)
		desc := get(1)
		amountStr := get(2)
		txType := strings.ToLower(get(3))
		categName := get(4)

		// Validate
		if dateStr == "" || amountStr == "" || txType == "" {
			res.Valid = false
			res.Error = "fecha, monto y tipo son requeridos"
			results = append(results, res)
			continue
		}

		date, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			res.Valid = false
			res.Error = fmt.Sprintf("fecha invalida: %q (usar YYYY-MM-DD)", dateStr)
			results = append(results, res)
			continue
		}

		amount, err := strconv.ParseFloat(strings.ReplaceAll(amountStr, ",", "."), 64)
		if err != nil || amount <= 0 {
			res.Valid = false
			res.Error = fmt.Sprintf("monto invalido: %q", amountStr)
			results = append(results, res)
			continue
		}

		if txType != "expense" && txType != "income" {
			res.Valid = false
			res.Error = fmt.Sprintf("tipo invalido: %q (usar expense o income)", txType)
			results = append(results, res)
			continue
		}

		var catID *uuid.UUID
		if categName != "" {
			if id, ok := catMap[strings.ToLower(categName)]; ok {
				catID = &id
			}
		}

		res.Valid = true
		res.Date = date.Format("2006-01-02")
		res.Desc = desc
		res.Amount = amount
		res.Type = txType
		res.Categ = categName

		if !dryRun {
			_, err := h.txSvc.Create(c.Request.Context(), services.CreateTransactionParams{
				WorkspaceID: wsID,
				UserID:      userID,
				CategoryID:  catID,
				Type:        txType,
				Amount:      amount,
				Description: desc,
				Date:        date,
			})
			if err != nil {
				res.Valid = false
				res.Error = "error al guardar"
			} else {
				imported++
			}
		}

		results = append(results, res)
	}

	skipped := 0
	for _, r := range results {
		if !r.Valid {
			skipped++
		}
	}
	if dryRun {
		imported = 0
	}

	response.OK(c, importSummary{
		Total:    len(results),
		Imported: imported,
		Skipped:  skipped,
		Rows:     results,
	})
}
