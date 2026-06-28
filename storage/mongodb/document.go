package mongodb

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/samber/oops"
)

const (
	SchemaVersion1 = "1.0.0"
	isoLayout      = "2006-01-02T15:04:05.000Z"
)

type ISOTime time.Time

func (t ISOTime) String() string {
	return time.Time(t).UTC().Format(isoLayout)
}

func (t ISOTime) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.String())
}

type DocumentFields struct {
	ID              string     `bson:"_id" json:"id"`
	SchemaVersion   string     `bson:"schema_version" json:"schema_version"`
	Revision        int64      `bson:"revision" json:"revision"`
	CreatedAt       time.Time  `bson:"created_at" json:"created_at"`
	UpdatedAt       time.Time  `bson:"updated_at" json:"updated_at"`
	CollectedAt     *time.Time `bson:"collected_at,omitempty" json:"collected_at,omitempty"`
	SourceUpdatedAt *time.Time `bson:"source_updated_at,omitempty" json:"source_updated_at,omitempty"`
	DeletedAt       *time.Time `bson:"deleted_at,omitempty" json:"deleted_at,omitempty"`
}

func NewDocumentFields(id string, schemaVersion string, now time.Time) (DocumentFields, error) {
	id = strings.TrimSpace(id)
	schemaVersion = strings.TrimSpace(schemaVersion)
	errb := oops.In("mongodb_document").With("id", id, "schema_version", schemaVersion)
	if id == "" {
		return DocumentFields{}, errb.New("mongodb document id is required")
	}
	if schemaVersion == "" {
		return DocumentFields{}, errb.New("mongodb document schema_version is required")
	}
	now = now.UTC().Truncate(time.Millisecond)
	return DocumentFields{
		ID:            id,
		SchemaVersion: schemaVersion,
		Revision:      1,
		CreatedAt:     now,
		UpdatedAt:     now,
	}, nil
}

func NewRevisionConflictError(collection string, id string, revision int64) error {
	return oops.In("mongodb_repository").
		With("collection", collection, "id", id, "revision", revision).
		Errorf("mongodb revision conflict in %s for %s at revision %d", collection, id, revision)
}
