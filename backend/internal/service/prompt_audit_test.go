package service

import (
	"reflect"
	"testing"

	"github.com/new-api-tools/backend/internal/config"
)

func TestSavePromptAuditConfigNormalizesAndVersions(t *testing.T) {
	t.Setenv("DATA_DIR", t.TempDir())
	config.Load()

	cfg, err := SavePromptAuditConfig(PromptAuditConfigInput{
		Enabled:  true,
		Keywords: []string{" Test_Sensitive ", "test_sensitive"},
		Raw:      "测试审查,ac-large-dict-test；博彩",
	})
	if err != nil {
		t.Fatalf("SavePromptAuditConfig returned error: %v", err)
	}

	want := []string{"test_sensitive", "测试审查", "ac-large-dict-test", "博彩"}
	if !reflect.DeepEqual(cfg.Keywords, want) {
		t.Fatalf("keywords mismatch:\nwant %#v\ngot  %#v", want, cfg.Keywords)
	}
	if !cfg.Enabled {
		t.Fatalf("expected config to be enabled")
	}
	if cfg.Version != 1 {
		t.Fatalf("expected version 1, got %d", cfg.Version)
	}

	loaded, err := GetPromptAuditConfig()
	if err != nil {
		t.Fatalf("GetPromptAuditConfig returned error: %v", err)
	}
	if !reflect.DeepEqual(loaded.Keywords, want) {
		t.Fatalf("loaded keywords mismatch:\nwant %#v\ngot  %#v", want, loaded.Keywords)
	}

	next, err := SavePromptAuditConfig(PromptAuditConfigInput{
		Enabled:  false,
		Keywords: []string{"next"},
	})
	if err != nil {
		t.Fatalf("second SavePromptAuditConfig returned error: %v", err)
	}
	if next.Version != 2 {
		t.Fatalf("expected version 2 after second save, got %d", next.Version)
	}
	if next.Enabled {
		t.Fatalf("expected second config to be disabled")
	}
}
