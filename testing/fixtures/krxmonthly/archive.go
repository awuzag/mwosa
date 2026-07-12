package krxmonthly

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	provider "github.com/awuzag/mwosa/providers/core"
	"github.com/awuzag/mwosa/storage/providerraw"
	"github.com/samber/oops"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

const (
	SchemaVersion  = 1
	maxArchiveFile = 32 << 20
)

type RawSnapshot struct {
	Provider         string          `json:"provider"`
	ProviderGroup    string          `json:"provider_group"`
	APIID            string          `json:"api_id"`
	BaseDate         string          `json:"base_date"`
	RowCount         int             `json:"row_count"`
	CanonicalSupport string          `json:"canonical_support"`
	Rows             json.RawMessage `json:"rows"`
}

type FileEntry struct {
	Path      string `json:"path"`
	Operation string `json:"operation"`
	BaseDate  string `json:"base_date"`
	RowCount  int    `json:"row_count"`
	SHA256    string `json:"sha256"`
}

type Manifest struct {
	SchemaVersion int         `json:"schema_version"`
	FixtureID     string      `json:"fixture_id"`
	Provider      string      `json:"provider"`
	From          string      `json:"from"`
	To            string      `json:"to"`
	CollectedAt   string      `json:"collected_at"`
	Operations    []string    `json:"operations"`
	TradingDates  []string    `json:"trading_dates"`
	SnapshotCount int         `json:"snapshot_count"`
	TotalRows     int         `json:"total_rows"`
	DatasetSHA256 string      `json:"dataset_sha256"`
	Files         []FileEntry `json:"files"`
}

type BuildOptions struct {
	FixtureID   string
	From        string
	To          string
	CollectedAt time.Time
	Overwrite   bool
}

type Dataset struct {
	Manifest  Manifest
	Snapshots []RawSnapshot
}

type LoadResult struct {
	Manifest Manifest                    `json:"manifest"`
	Bulk     providerraw.BulkWriteResult `json:"bulk"`
}

type archiveFile struct {
	entry FileEntry
	data  []byte
}

func WriteArchive(path string, opts BuildOptions, snapshots []RawSnapshot) (Manifest, error) {
	errB := oops.In("krx_monthly_fixture_archive").With("path", path, "snapshot_count", len(snapshots))
	path = strings.TrimSpace(path)
	if path == "" {
		return Manifest{}, errB.New("archive path is required")
	}
	if strings.TrimSpace(opts.FixtureID) == "" {
		return Manifest{}, errB.New("fixture id is required")
	}
	if len(snapshots) == 0 {
		return Manifest{}, errB.New("at least one KRX snapshot is required")
	}
	if _, err := os.Stat(path); err == nil && !opts.Overwrite {
		return Manifest{}, errB.New("archive already exists; set overwrite to replace it")
	} else if err != nil && !os.IsNotExist(err) {
		return Manifest{}, errB.Wrapf(err, "inspect archive path")
	}

	sorted := append([]RawSnapshot(nil), snapshots...)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].BaseDate == sorted[j].BaseDate {
			return sorted[i].APIID < sorted[j].APIID
		}
		return sorted[i].BaseDate < sorted[j].BaseDate
	})

	files := make([]archiveFile, 0, len(sorted))
	operations := map[string]struct{}{}
	tradingDates := map[string]struct{}{}
	totalRows := 0
	for index, snapshot := range sorted {
		if err := validateSnapshot(snapshot); err != nil {
			return Manifest{}, errB.With("index", index).Wrap(err)
		}
		data, err := json.MarshalIndent(snapshot, "", "  ")
		if err != nil {
			return Manifest{}, errB.With("index", index).Wrapf(err, "encode KRX snapshot")
		}
		data = append(data, '\n')
		checksum := sha256.Sum256(data)
		entry := FileEntry{
			Path:      "data/" + snapshot.APIID + "/" + snapshot.BaseDate + ".json",
			Operation: snapshot.APIID,
			BaseDate:  snapshot.BaseDate,
			RowCount:  snapshot.RowCount,
			SHA256:    hex.EncodeToString(checksum[:]),
		}
		files = append(files, archiveFile{entry: entry, data: data})
		operations[snapshot.APIID] = struct{}{}
		tradingDates[snapshot.BaseDate] = struct{}{}
		totalRows += snapshot.RowCount
	}

	manifest := Manifest{
		SchemaVersion: SchemaVersion,
		FixtureID:     strings.TrimSpace(opts.FixtureID),
		Provider:      "krx",
		From:          strings.TrimSpace(opts.From),
		To:            strings.TrimSpace(opts.To),
		CollectedAt:   opts.CollectedAt.UTC().Format(time.RFC3339),
		Operations:    sortedKeys(operations),
		TradingDates:  sortedKeys(tradingDates),
		SnapshotCount: len(files),
		TotalRows:     totalRows,
		Files:         make([]FileEntry, 0, len(files)),
	}
	for _, file := range files {
		manifest.Files = append(manifest.Files, file.entry)
	}
	manifest.DatasetSHA256 = datasetChecksum(manifest.Files)

	if err := writeZip(path, manifest, files); err != nil {
		return Manifest{}, errB.Wrap(err)
	}
	return manifest, nil
}

func ReadArchive(path string) (Dataset, error) {
	errB := oops.In("krx_monthly_fixture_archive").With("path", path)
	reader, err := zip.OpenReader(path)
	if err != nil {
		return Dataset{}, errB.Wrapf(err, "open KRX fixture archive")
	}
	defer reader.Close()

	files := make(map[string]*zip.File, len(reader.File))
	for _, file := range reader.File {
		files[file.Name] = file
	}
	manifestFile := files["manifest.json"]
	if manifestFile == nil {
		return Dataset{}, errB.New("KRX fixture archive is missing manifest.json")
	}
	manifestBytes, err := readZipFile(manifestFile)
	if err != nil {
		return Dataset{}, errB.Wrap(err)
	}
	var manifest Manifest
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		return Dataset{}, errB.Wrapf(err, "decode KRX fixture manifest")
	}
	if err := validateManifest(manifest); err != nil {
		return Dataset{}, errB.Wrap(err)
	}

	snapshots := make([]RawSnapshot, 0, len(manifest.Files))
	totalRows := 0
	for _, entry := range manifest.Files {
		file := files[entry.Path]
		if file == nil {
			return Dataset{}, errB.With("file", entry.Path).New("KRX fixture archive file is missing")
		}
		data, err := readZipFile(file)
		if err != nil {
			return Dataset{}, errB.With("file", entry.Path).Wrap(err)
		}
		checksum := sha256.Sum256(data)
		if hex.EncodeToString(checksum[:]) != entry.SHA256 {
			return Dataset{}, errB.With("file", entry.Path).New("KRX fixture file checksum mismatch")
		}
		var snapshot RawSnapshot
		if err := json.Unmarshal(data, &snapshot); err != nil {
			return Dataset{}, errB.With("file", entry.Path).Wrapf(err, "decode KRX fixture snapshot")
		}
		if err := validateSnapshot(snapshot); err != nil {
			return Dataset{}, errB.With("file", entry.Path).Wrap(err)
		}
		if snapshot.APIID != entry.Operation || snapshot.BaseDate != entry.BaseDate || snapshot.RowCount != entry.RowCount {
			return Dataset{}, errB.With("file", entry.Path).New("KRX fixture manifest entry does not match snapshot")
		}
		snapshots = append(snapshots, snapshot)
		totalRows += snapshot.RowCount
	}
	if totalRows != manifest.TotalRows || len(snapshots) != manifest.SnapshotCount {
		return Dataset{}, errB.New("KRX fixture manifest totals do not match archive data")
	}
	if datasetChecksum(manifest.Files) != manifest.DatasetSHA256 {
		return Dataset{}, errB.New("KRX fixture dataset checksum mismatch")
	}
	return Dataset{Manifest: manifest, Snapshots: snapshots}, nil
}

func LoadArchive(ctx context.Context, database *mongo.Database, path string) (LoadResult, error) {
	errB := oops.In("krx_monthly_fixture_loader").With("path", path)
	if database == nil {
		return LoadResult{}, errB.New("MongoDB database is required")
	}
	dataset, err := ReadArchive(path)
	if err != nil {
		return LoadResult{}, errB.Wrap(err)
	}
	snapshots := make([]providerraw.Snapshot, 0, len(dataset.Snapshots))
	for _, raw := range dataset.Snapshots {
		var rows any
		if err := json.Unmarshal(raw.Rows, &rows); err != nil {
			return LoadResult{}, errB.With("operation", raw.APIID, "base_date", raw.BaseDate).Wrapf(err, "decode KRX rows")
		}
		snapshots = append(snapshots, providerraw.Snapshot{
			Provider:         provider.ProviderID(raw.Provider),
			Group:            provider.GroupID(raw.ProviderGroup),
			Operation:        provider.OperationID(raw.APIID),
			BaseDate:         raw.BaseDate,
			CanonicalSupport: raw.CanonicalSupport,
			Rows:             rows,
			RowCount:         raw.RowCount,
		})
	}
	repository, err := providerraw.NewMongoRepository(database)
	if err != nil {
		return LoadResult{}, errB.Wrap(err)
	}
	bulk, err := repository.UpsertSnapshots(ctx, snapshots)
	if err != nil {
		return LoadResult{}, errB.Wrap(err)
	}
	return LoadResult{Manifest: dataset.Manifest, Bulk: bulk}, nil
}

func validateSnapshot(snapshot RawSnapshot) error {
	errB := oops.In("krx_monthly_fixture_snapshot").With("operation", snapshot.APIID, "base_date", snapshot.BaseDate)
	if snapshot.Provider != "krx" || strings.TrimSpace(snapshot.ProviderGroup) == "" || strings.TrimSpace(snapshot.APIID) == "" {
		return errB.New("KRX fixture snapshot has an invalid source")
	}
	if _, err := time.Parse("2006-01-02", snapshot.BaseDate); err != nil {
		return errB.Wrapf(err, "parse KRX fixture base date")
	}
	if !json.Valid(snapshot.Rows) {
		return errB.New("KRX fixture rows are invalid JSON")
	}
	var rows []json.RawMessage
	if err := json.Unmarshal(snapshot.Rows, &rows); err != nil {
		return errB.Wrapf(err, "decode KRX fixture rows")
	}
	if len(rows) != snapshot.RowCount {
		return errB.With("row_count", snapshot.RowCount, "decoded_rows", len(rows)).New("KRX fixture row count mismatch")
	}
	return nil
}

func validateManifest(manifest Manifest) error {
	errB := oops.In("krx_monthly_fixture_manifest").With("fixture_id", manifest.FixtureID)
	if manifest.SchemaVersion != SchemaVersion {
		return errB.With("schema_version", manifest.SchemaVersion).New("unsupported KRX fixture schema version")
	}
	if strings.TrimSpace(manifest.FixtureID) == "" || manifest.Provider != "krx" {
		return errB.New("KRX fixture manifest identity is invalid")
	}
	if len(manifest.Files) == 0 || manifest.SnapshotCount != len(manifest.Files) {
		return errB.New("KRX fixture manifest files are invalid")
	}
	return nil
}

func writeZip(path string, manifest Manifest, files []archiveFile) (returnErr error) {
	errB := oops.In("krx_monthly_fixture_archive").With("path", path)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return errB.Wrapf(err, "create fixture archive directory")
	}
	temporary, err := os.CreateTemp(filepath.Dir(path), ".krx-monthly-*.zip")
	if err != nil {
		return errB.Wrapf(err, "create temporary fixture archive")
	}
	temporaryPath := temporary.Name()
	defer func() {
		if returnErr != nil {
			_ = os.Remove(temporaryPath)
		}
	}()

	archive := zip.NewWriter(temporary)
	manifestBytes, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return errB.Wrapf(err, "encode KRX fixture manifest")
	}
	if err := writeZipEntry(archive, "manifest.json", append(manifestBytes, '\n')); err != nil {
		return errB.Wrap(err)
	}
	for _, file := range files {
		if err := writeZipEntry(archive, file.entry.Path, file.data); err != nil {
			return errB.Wrap(err)
		}
	}
	if err := archive.Close(); err != nil {
		return errB.Wrapf(err, "close KRX fixture archive")
	}
	if err := temporary.Close(); err != nil {
		return errB.Wrapf(err, "close temporary fixture archive")
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return errB.Wrapf(err, "replace KRX fixture archive")
	}
	return nil
}

func writeZipEntry(archive *zip.Writer, name string, data []byte) error {
	header := &zip.FileHeader{Name: name, Method: zip.Deflate}
	header.Modified = time.Date(1980, 1, 1, 0, 0, 0, 0, time.UTC)
	header.SetMode(0o644)
	writer, err := archive.CreateHeader(header)
	if err != nil {
		return oops.In("krx_monthly_fixture_archive").With("file", name).Wrapf(err, "create ZIP entry")
	}
	if _, err := io.Copy(writer, bytes.NewReader(data)); err != nil {
		return oops.In("krx_monthly_fixture_archive").With("file", name).Wrapf(err, "write ZIP entry")
	}
	return nil
}

func readZipFile(file *zip.File) ([]byte, error) {
	if file.UncompressedSize64 > maxArchiveFile {
		return nil, oops.In("krx_monthly_fixture_archive").With("file", file.Name).New("ZIP entry exceeds size limit")
	}
	reader, err := file.Open()
	if err != nil {
		return nil, oops.In("krx_monthly_fixture_archive").With("file", file.Name).Wrapf(err, "open ZIP entry")
	}
	defer reader.Close()
	data, err := io.ReadAll(io.LimitReader(reader, maxArchiveFile+1))
	if err != nil {
		return nil, oops.In("krx_monthly_fixture_archive").With("file", file.Name).Wrapf(err, "read ZIP entry")
	}
	if len(data) > maxArchiveFile {
		return nil, oops.In("krx_monthly_fixture_archive").With("file", file.Name).New("ZIP entry exceeds size limit")
	}
	return data, nil
}

func datasetChecksum(entries []FileEntry) string {
	hash := sha256.New()
	for _, entry := range entries {
		_, _ = hash.Write([]byte(entry.Path))
		_, _ = hash.Write([]byte{0})
		_, _ = hash.Write([]byte(entry.SHA256))
		_, _ = hash.Write([]byte{0})
	}
	return hex.EncodeToString(hash.Sum(nil))
}

func sortedKeys(values map[string]struct{}) []string {
	out := make([]string, 0, len(values))
	for value := range values {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}
