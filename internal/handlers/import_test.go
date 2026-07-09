package handlers

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/andrespalacio/finapp-backend/internal/middleware"
	"github.com/andrespalacio/finapp-backend/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"
)

type mockImportTransactionService struct {
	createFn func(ctx context.Context, p services.CreateTransactionParams) (services.TransactionView, error)
}

func (m *mockImportTransactionService) Create(ctx context.Context, p services.CreateTransactionParams) (services.TransactionView, error) {
	return m.createFn(ctx, p)
}

type mockImportCategoryService struct {
	listForWorkspaceFn func(ctx context.Context, workspaceID uuid.UUID) ([]services.CategoryView, error)
}

func (m *mockImportCategoryService) ListForWorkspace(ctx context.Context, workspaceID uuid.UUID) ([]services.CategoryView, error) {
	return m.listForWorkspaceFn(ctx, workspaceID)
}

func newImportRouter(txSvc *mockImportTransactionService, catSvc *mockImportCategoryService, userID uuid.UUID) *gin.Engine {
	gin.SetMode(gin.TestMode)
	h := NewImportHandler(txSvc, catSvc)
	r := gin.New()

	wsGroup := r.Group("/workspaces/:workspace_id", withUserID(userID), middleware.WorkspaceMiddleware(stubMemberChecker{isMember: true}))
	{
		wsGroup.GET("/transactions/import/template", h.Template)
		wsGroup.POST("/transactions/import", h.Import)
	}
	return r
}

// buildXLSX writes rows (including a header row at index 0) to an in-memory xlsx file.
func buildXLSX(t *testing.T, rows [][]string) []byte {
	t.Helper()
	f := excelize.NewFile()
	defer f.Close() //nolint:errcheck
	sheet := f.GetSheetName(0)
	for i, row := range rows {
		for j, val := range row {
			cell, _ := excelize.CoordinatesToCellName(j+1, i+1)
			f.SetCellValue(sheet, cell, val)
		}
	}
	var buf bytes.Buffer
	require.NoError(t, f.Write(&buf))
	return buf.Bytes()
}

func multipartFileRequest(t *testing.T, url string, filename string, content []byte) *http.Request {
	t.Helper()
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	part, err := w.CreateFormFile("file", filename)
	require.NoError(t, err)
	_, err = part.Write(content)
	require.NoError(t, err)
	require.NoError(t, w.Close())

	req := httptest.NewRequest(http.MethodPost, url, &body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	return req
}

func TestImportHandler_Template(t *testing.T) {
	userID := uuid.New()
	wsID := uuid.New()
	r := newImportRouter(&mockImportTransactionService{}, &mockImportCategoryService{}, userID)

	req := httptest.NewRequest(http.MethodGet, "/workspaces/"+wsID.String()+"/transactions/import/template", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotEmpty(t, w.Body.Bytes())
	assert.Contains(t, w.Header().Get("Content-Disposition"), "plantilla_transacciones.xlsx")
}

func TestImportHandler_Import_MissingFile(t *testing.T) {
	userID := uuid.New()
	wsID := uuid.New()
	r := newImportRouter(&mockImportTransactionService{}, &mockImportCategoryService{}, userID)

	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	require.NoError(t, w.Close())
	req := httptest.NewRequest(http.MethodPost, "/workspaces/"+wsID.String()+"/transactions/import", &body)
	req.Header.Set("Content-Type", w.FormDataContentType())

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestImportHandler_Import_MalformedFile(t *testing.T) {
	userID := uuid.New()
	wsID := uuid.New()
	r := newImportRouter(&mockImportTransactionService{}, &mockImportCategoryService{}, userID)

	req := multipartFileRequest(t, "/workspaces/"+wsID.String()+"/transactions/import", "bad.xlsx", []byte("not a real xlsx file"))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestImportHandler_Import_ValidRows(t *testing.T) {
	userID := uuid.New()
	wsID := uuid.New()
	created := 0
	txSvc := &mockImportTransactionService{
		createFn: func(_ context.Context, _ services.CreateTransactionParams) (services.TransactionView, error) {
			created++
			return services.TransactionView{}, nil
		},
	}
	catSvc := &mockImportCategoryService{
		listForWorkspaceFn: func(_ context.Context, _ uuid.UUID) ([]services.CategoryView, error) {
			return []services.CategoryView{{ID: uuid.New(), Name: "Alimentacion"}}, nil
		},
	}
	r := newImportRouter(txSvc, catSvc, userID)

	rows := [][]string{
		{"fecha", "descripcion", "monto", "tipo", "categoria", "notas"},
		{"2026-01-15", "Mercado", "50000", "expense", "Alimentacion", "compra"},
		{"2026-01-16", "Salario", "1000000", "income", "", ""},
	}
	xlsx := buildXLSX(t, rows)

	req := multipartFileRequest(t, "/workspaces/"+wsID.String()+"/transactions/import", "data.xlsx", xlsx)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 2, created)
	assert.Contains(t, w.Body.String(), `"imported":2`)
}

func TestImportHandler_Import_DryRun(t *testing.T) {
	userID := uuid.New()
	wsID := uuid.New()
	created := 0
	txSvc := &mockImportTransactionService{
		createFn: func(_ context.Context, _ services.CreateTransactionParams) (services.TransactionView, error) {
			created++
			return services.TransactionView{}, nil
		},
	}
	catSvc := &mockImportCategoryService{
		listForWorkspaceFn: func(_ context.Context, _ uuid.UUID) ([]services.CategoryView, error) { return nil, nil },
	}
	r := newImportRouter(txSvc, catSvc, userID)

	rows := [][]string{
		{"fecha", "descripcion", "monto", "tipo", "categoria", "notas"},
		{"2026-01-15", "Mercado", "50000", "expense", "", ""},
	}
	xlsx := buildXLSX(t, rows)

	req := multipartFileRequest(t, "/workspaces/"+wsID.String()+"/transactions/import?dry_run=true", "data.xlsx", xlsx)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 0, created, "dry_run must not call the transaction service")
	assert.Contains(t, w.Body.String(), `"imported":0`)
}

func TestImportHandler_Import_InvalidRowSkipped(t *testing.T) {
	userID := uuid.New()
	wsID := uuid.New()
	txSvc := &mockImportTransactionService{
		createFn: func(_ context.Context, _ services.CreateTransactionParams) (services.TransactionView, error) {
			return services.TransactionView{}, nil
		},
	}
	catSvc := &mockImportCategoryService{
		listForWorkspaceFn: func(_ context.Context, _ uuid.UUID) ([]services.CategoryView, error) { return nil, nil },
	}
	r := newImportRouter(txSvc, catSvc, userID)

	rows := [][]string{
		{"fecha", "descripcion", "monto", "tipo", "categoria", "notas"},
		{"not-a-date", "Mercado", "50000", "expense", "", ""},
	}
	xlsx := buildXLSX(t, rows)

	req := multipartFileRequest(t, "/workspaces/"+wsID.String()+"/transactions/import", "data.xlsx", xlsx)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"skipped":1`)
}
