package tests

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/azzurrotech/song/internal/handler"
)

// ... existing tests ...

func TestFormBuilderRoute(t *testing.T) {
	req, err := http.NewRequest("GET", "/formbuilder", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("FormBuilder returned wrong status: got %v want %v", status, http.StatusOK)
	}

	if !strings.Contains(rr.Body.String(), "SONG FormBuilder") {
		t.Error("FormBuilder page content missing")
	}
}

func TestSaveFormDefinition(t *testing.T) {
	jsonBody := `{"slug":"test-form","title":"Test Form","fields":[]}`
	req, err := http.NewRequest("POST", "/api/forms", strings.NewReader(jsonBody))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("SaveForm returned wrong status: got %v want %v", status, http.StatusOK)
	}
}

func TestRenderDynamicForm(t *testing.T) {
	req, err := http.NewRequest("GET", "/forms/contact", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("DynamicForm returned wrong status: got %v want %v", status, http.StatusOK)
	}

	if !strings.Contains(rr.Body.String(), "Dynamic Form") {
		t.Error("Dynamic form content missing")
	}
}
