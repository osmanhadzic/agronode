package models

import (
	"reflect"
	"sync"
	"testing"

	"gorm.io/gorm/schema"
)

func parseDeviceSchema(t *testing.T) *schema.Schema {
	t.Helper()

	parsed, err := schema.Parse(&Device{}, &sync.Map{}, schema.NamingStrategy{})
	if err != nil {
		t.Fatalf("parse device schema: %v", err)
	}

	return parsed
}

func TestDevice_HardwareIDMapping(t *testing.T) {
	parsed := parseDeviceSchema(t)

	field, ok := parsed.FieldsByName["HardwareID"]
	if !ok {
		t.Fatal("Device has no HardwareID field")
	}

	if field.DBName != "hardware_id" {
		t.Fatalf("expected column hardware_id, got %q", field.DBName)
	}

	if field.Size != 64 {
		t.Fatalf("expected size 64, got %d", field.Size)
	}

	if field.FieldType.Kind() != reflect.Ptr || field.FieldType.Elem().Kind() != reflect.String {
		t.Fatalf("expected *string so missing IDs persist as NULL, got %s", field.FieldType)
	}

	if parsed.Table != "devices" {
		t.Fatalf("expected table devices, got %q", parsed.Table)
	}
}

func TestDevice_HardwareIDDeclaresNoGormIndex(t *testing.T) {
	// The partial unique index is owned by migration 000024; a GORM-declared
	// index would compete with it if AutoMigrate were ever enabled.
	for _, index := range parseDeviceSchema(t).ParseIndexes() {
		for _, indexField := range index.Fields {
			if indexField.DBName == "hardware_id" {
				t.Fatalf("hardware_id must not declare a GORM index, found %q", index.Name)
			}
		}
	}
}
