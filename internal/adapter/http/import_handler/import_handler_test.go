package http

import (
	"context"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/DovCaio/saldo/internal/transaction"
)

type fakeImportUseCase struct {
	called bool
	err    error
}

func (f *fakeImportUseCase) Execute(
	ctx context.Context,
	r io.Reader,
) error {
	f.called = true
	return f.err
}

func TestImportHandler_Success(t *testing.T) {
	fake := &fakeImportUseCase{}
	handler := NewImportHandler(fake)

	var body strings.Builder
	writer := multipart.NewWriter(&body)

	fileWriter, err := writer.CreateFormFile("file", "transacoes.csv")
	if err != nil {
		t.Fatal(err)
	}

	csv := "date,description,amount,type\n" +
		"2026-10-01,Salario,5000.00,INCOME\n"

	if _, err := io.Copy(fileWriter, strings.NewReader(csv)); err != nil {
		t.Fatal(err)
	}

	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(
		http.MethodPost,
		"/transactions/import",
		strings.NewReader(body.String()),
	)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if !fake.called {
		t.Fatal("expected use case to be called")
	}

	if rec.Code != http.StatusNoContent {
		t.Errorf(
			"expected status %d, got %d",
			http.StatusNoContent,
			rec.Code,
		)
	}
}

func TestImportHandler_UseCaseError(t *testing.T) {
	fake := &fakeImportUseCase{
		err: transaction.ErrDuplicateInBatch,
	}
	handler := NewImportHandler(fake)

	var body strings.Builder
	writer := multipart.NewWriter(&body)

	fileWriter, err := writer.CreateFormFile("file", "transacoes.csv")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := io.WriteString(
		fileWriter,
		"date,description,amount,type\n",
	); err != nil {
		t.Fatal(err)
	}

	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(
		http.MethodPost,
		"/transactions/import",
		strings.NewReader(body.String()),
	)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if !fake.called {
		t.Fatal("expected use case to be called")
	}

	if rec.Code != http.StatusConflict {
		t.Errorf(
			"expected status %d, got %d",
			http.StatusConflict,
			rec.Code,
		)
	}
}

func newMultipartRequest(t *testing.T, fieldName, filename, contents string) *http.Request {
	t.Helper()

	var body strings.Builder
	writer := multipart.NewWriter(&body)

	if fieldName != "" {
		fileWriter, err := writer.CreateFormFile(fieldName, filename)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := io.WriteString(fileWriter, contents); err != nil {
			t.Fatal(err)
		}
	}

	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(
		http.MethodPost,
		"/transactions/import",
		strings.NewReader(body.String()),
	)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req
}

func TestImportHandler_MissingFile(t *testing.T) {
	fake := &fakeImportUseCase{}
	handler := NewImportHandler(fake)

	req := newMultipartRequest(t, "", "", "")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	if fake.called {
		t.Error("use case should not be called")
	}
}

func TestImportHandler_InvalidMultipart(t *testing.T) {
	fake := &fakeImportUseCase{}
	handler := NewImportHandler(fake)

	req := httptest.NewRequest(http.MethodPost, "/transactions/import", strings.NewReader("not multipart"))
	req.Header.Set("Content-Type", "text/plain")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestImportHandler_FileTooLarge(t *testing.T) {
	fake := &fakeImportUseCase{}
	handler := NewImportHandler(fake)

	body := strings.Repeat("a", int(maxUploadSize)+1024)
	req := httptest.NewRequest(http.MethodPost, "/transactions/import", strings.NewReader(body))
	req.Header.Set("Content-Type", "multipart/form-data; boundary=----boundary")
	req.ContentLength = int64(len(body))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge && rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 413 or 400", rec.Code)
	}
}

func TestImportHandler_MapsDomainErrors(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
	}{
		{name: "empty import", err: transaction.ErrEmptyImport, wantStatus: http.StatusBadRequest},
		{name: "invalid date", err: transaction.ErrInvalidDate, wantStatus: http.StatusBadRequest},
		{name: "empty description", err: transaction.ErrEmptyDescription, wantStatus: http.StatusBadRequest},
		{name: "invalid amount", err: transaction.ErrInvalidAmount, wantStatus: http.StatusBadRequest},
		{name: "invalid type", err: transaction.ErrInvalidType, wantStatus: http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeImportUseCase{err: tt.err}
			handler := NewImportHandler(fake)
			req := newMultipartRequest(t, "file", "transacoes.csv", "date,description,amount,type\n")
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
		})
	}
}
