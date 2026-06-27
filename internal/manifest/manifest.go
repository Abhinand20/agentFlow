// Package manifest records per-source build ownership for generated host artifacts.
package manifest

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Abhinand20/agentFlow/internal/emit"
)

const (
	// SchemaVersion is the manifest JSON schema version.
	SchemaVersion = 1
	// HistoryCap is the maximum number of build records kept in history.
	HistoryCap = 20
	toolName   = "af"
)

// Manifest records the current build and capped history for one .af source.
type Manifest struct {
	SchemaVersion int          `json:"schemaVersion"`
	Tool          string       `json:"tool"`
	ToolVersion   string       `json:"toolVersion"`
	Target        string       `json:"target"`
	Version       int          `json:"version"`
	GeneratedAt   string       `json:"generatedAt"`
	Source        Source       `json:"source"`
	IRHash        string       `json:"irHash"`
	Artifacts     []Artifact   `json:"artifacts"`
	History       []BuildRecord `json:"history"`
}

// Source identifies the .af file that produced a build.
type Source struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

// Artifact is one generated host file owned by a build.
type Artifact struct {
	Path   string `json:"path"`
	Role   string `json:"role"`
	SHA256 string `json:"sha256"`
	Bytes  int    `json:"bytes"`
}

// BuildRecord is one entry in the capped build history.
type BuildRecord struct {
	Version        int                `json:"version"`
	GeneratedAt    string             `json:"generatedAt"`
	IRHash         string             `json:"irHash"`
	SourceSHA256   string             `json:"sourceSha256"`
	Artifacts      []HistoryArtifact  `json:"artifacts"`
}

// HistoryArtifact is the slim artifact record stored in history.
type HistoryArtifact struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

// BuildOptions configures manifest construction from a binding emit FS.
type BuildOptions struct {
	Target         string
	SourcePath     string
	SourceSHA256   string
	IRHash         string
	FS             *emit.FS
	Prior          *Manifest
	Now            func() time.Time
	ToolVersion    string
}

// Changes describes artifact differences between two build records.
type Changes struct {
	Added    []string
	Removed  []string
	Changed  []string
}

// DriftReport classifies on-disk artifacts vs the manifest record.
type DriftReport struct {
	Clean      []Artifact
	Modified   []Artifact
	Missing    []Artifact
	Unreadable []Artifact
}

// TargetRoot returns the host config root directory for a binding target.
func TargetRoot(target string) string {
	switch target {
	case "cursor":
		return ".cursor"
	case "claude-code":
		return ".claude"
	default:
		return "." + target
	}
}

// ManifestsDir returns the manifests directory relative to the output root.
func ManifestsDir(target string) string {
	return filepath.ToSlash(filepath.Join(TargetRoot(target), ".agentflow", "manifests"))
}

// Slug derives a stable per-source filename stem from a source path.
func Slug(sourcePath string) string {
	abs, err := filepath.Abs(sourcePath)
	if err != nil {
		abs = sourcePath
	}
	abs = filepath.Clean(abs)
	sum := sha256.Sum256([]byte(abs))
	short := hex.EncodeToString(sum[:])[:8]
	base := strings.TrimSuffix(filepath.Base(abs), ".af")
	if base == "" {
		base = "source"
	}
	return base + "-" + short
}

// ManifestRelPath returns the manifest path relative to the output root.
func ManifestRelPath(target, sourcePath string) string {
	return filepath.ToSlash(filepath.Join(ManifestsDir(target), Slug(sourcePath)+".json"))
}

// HashBytes returns the lowercase hex sha256 of content.
func HashBytes(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}

// HashSource returns sha256 hex of source text.
func HashSource(source string) string {
	return HashBytes([]byte(source))
}

// Build constructs a manifest from binding output and prior state.
func Build(opts BuildOptions) Manifest {
	now := opts.Now
	if now == nil {
		now = time.Now
	}
	toolVersion := opts.ToolVersion
	if toolVersion == "" {
		toolVersion = "dev"
	}

	manifestsPrefix := ManifestsDir(opts.Target) + "/"
	artifacts := collectArtifacts(opts.FS, manifestsPrefix)

	version := 1
	var history []BuildRecord
	if opts.Prior != nil {
		version = opts.Prior.Version + 1
		history = append(history, opts.Prior.History...)
	}

	generatedAt := now().UTC().Format(time.RFC3339)
	record := BuildRecord{
		Version:      version,
		GeneratedAt:  generatedAt,
		IRHash:       opts.IRHash,
		SourceSHA256: opts.SourceSHA256,
		Artifacts:    slimArtifacts(artifacts),
	}
	history = prependHistory(history, record)

	return Manifest{
		SchemaVersion: SchemaVersion,
		Tool:          toolName,
		ToolVersion:   toolVersion,
		Target:        opts.Target,
		Version:       version,
		GeneratedAt:   generatedAt,
		Source: Source{
			Path:   opts.SourcePath,
			SHA256: opts.SourceSHA256,
		},
		IRHash:    opts.IRHash,
		Artifacts: artifacts,
		History:   history,
	}
}

func collectArtifacts(hostFS *emit.FS, manifestsPrefix string) []Artifact {
	if hostFS == nil {
		return nil
	}
	var artifacts []Artifact
	for _, path := range hostFS.Paths() {
		if strings.HasPrefix(path, manifestsPrefix) {
			continue
		}
		content, ok := hostFS.Get(path)
		if !ok {
			continue
		}
		artifacts = append(artifacts, Artifact{
			Path:   path,
			Role:   classifyRole(path),
			SHA256: HashBytes(content),
			Bytes:  len(content),
		})
	}
	sort.Slice(artifacts, func(i, j int) bool {
		return artifacts[i].Path < artifacts[j].Path
	})
	return artifacts
}

func classifyRole(path string) string {
	path = filepath.ToSlash(path)
	switch {
	case strings.Contains(path, "/agents/"):
		return "agent"
	case strings.Contains(path, "/commands/"):
		return "command"
	case strings.HasSuffix(path, "mcp.json"):
		return "mcp"
	case strings.Contains(path, "/hooks"):
		return "hooks"
	default:
		return "other"
	}
}

func slimArtifacts(artifacts []Artifact) []HistoryArtifact {
	out := make([]HistoryArtifact, len(artifacts))
	for i, a := range artifacts {
		out[i] = HistoryArtifact{Path: a.Path, SHA256: a.SHA256}
	}
	return out
}

func prependHistory(history []BuildRecord, record BuildRecord) []BuildRecord {
	out := make([]BuildRecord, 0, len(history)+1)
	out = append(out, record)
	out = append(out, history...)
	if len(out) > HistoryCap {
		out = out[:HistoryCap]
	}
	return out
}

// Marshal serializes a manifest to indented JSON.
func Marshal(m Manifest) ([]byte, error) {
	return json.MarshalIndent(m, "", "  ")
}

// Unmarshal loads a manifest from JSON.
func Unmarshal(data []byte) (Manifest, error) {
	var m Manifest
	err := json.Unmarshal(data, &m)
	return m, err
}

// Load reads the manifest for one source from dir.
func Load(dir, target, sourcePath string) (*Manifest, bool, error) {
	path := filepath.Join(dir, filepath.FromSlash(ManifestRelPath(target, sourcePath)))
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, false, nil
		}
		return nil, false, err
	}
	m, err := Unmarshal(data)
	if err != nil {
		return nil, false, fmt.Errorf("manifest: decode %s: %w", path, err)
	}
	return &m, true, nil
}

// LoadAll reads every manifest under the target manifests directory.
// Invalid JSON files are skipped; when warn is non-nil it receives AF313 messages.
func LoadAll(dir, target string, warn func(string)) ([]*Manifest, error) {
	manifestsDir := filepath.Join(dir, filepath.FromSlash(ManifestsDir(target)))
	entries, err := os.ReadDir(manifestsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var out []*Manifest
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(manifestsDir, entry.Name()))
		if err != nil {
			return nil, err
		}
		m, err := Unmarshal(data)
		if err != nil {
			if warn != nil {
				warn(fmt.Sprintf("warning AF313: skip invalid manifest %s: %v", entry.Name(), err))
			}
			continue
		}
		cp := m
		out = append(out, &cp)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Source.Path < out[j].Source.Path
	})
	return out, nil
}

// Overlaps returns artifact paths owned by other manifests that intersect with m.
func Overlaps(m *Manifest, others []*Manifest) map[string]string {
	if m == nil {
		return nil
	}
	owned := make(map[string]string, len(m.Artifacts))
	for _, a := range m.Artifacts {
		owned[a.Path] = m.Source.Path
	}
	conflicts := make(map[string]string)
	for _, other := range others {
		if other == nil || other.Source.Path == m.Source.Path {
			continue
		}
		for _, a := range other.Artifacts {
			if owner, ok := owned[a.Path]; ok {
				conflicts[a.Path] = owner + " vs " + other.Source.Path
			}
		}
	}
	return conflicts
}

// ArtifactOwners maps artifact path to every source path that claims ownership.
func ArtifactOwners(manifests []*Manifest) map[string][]string {
	owners := make(map[string][]string)
	for _, m := range manifests {
		if m == nil {
			continue
		}
		for _, a := range m.Artifacts {
			owners[a.Path] = appendUniqueSource(owners[a.Path], m.Source.Path)
		}
	}
	return owners
}

// OtherOwner returns another source that owns path, if any.
func OtherOwner(path, source string, owners map[string][]string) (string, bool) {
	for _, owner := range owners[path] {
		if owner != source {
			return owner, true
		}
	}
	return "", false
}

func appendUniqueSource(sources []string, source string) []string {
	for _, s := range sources {
		if s == source {
			return sources
		}
	}
	return append(sources, source)
}

// CurrentRecord returns the current build as a BuildRecord.
func (m *Manifest) CurrentRecord() BuildRecord {
	if m == nil {
		return BuildRecord{}
	}
	return BuildRecord{
		Version:      m.Version,
		GeneratedAt:  m.GeneratedAt,
		IRHash:       m.IRHash,
		SourceSHA256: m.Source.SHA256,
		Artifacts:    slimArtifacts(m.Artifacts),
	}
}

// FindRecord returns the history record with the given version, or nil.
func (m *Manifest) FindRecord(version int) *BuildRecord {
	if m == nil {
		return nil
	}
	if m.Version == version {
		rec := m.CurrentRecord()
		return &rec
	}
	for i := range m.History {
		if m.History[i].Version == version {
			rec := m.History[i]
			return &rec
		}
	}
	return nil
}

// Diff compares artifact sets between two build records.
func Diff(a, b BuildRecord) Changes {
	aMap := artifactMap(a.Artifacts)
	bMap := artifactMap(b.Artifacts)

	var added, removed, changed []string
	for path, sha := range bMap {
		prev, ok := aMap[path]
		if !ok {
			added = append(added, path)
			continue
		}
		if prev != sha {
			changed = append(changed, path)
		}
	}
	for path := range aMap {
		if _, ok := bMap[path]; !ok {
			removed = append(removed, path)
		}
	}
	sort.Strings(added)
	sort.Strings(removed)
	sort.Strings(changed)
	return Changes{Added: added, Removed: removed, Changed: changed}
}

func artifactMap(artifacts []HistoryArtifact) map[string]string {
	out := make(map[string]string, len(artifacts))
	for _, a := range artifacts {
		out[a.Path] = a.SHA256
	}
	return out
}

// DriftCheck compares manifest artifacts to on-disk files under dir.
func DriftCheck(m *Manifest, dir string) DriftReport {
	var report DriftReport
	if m == nil {
		return report
	}
	for _, art := range m.Artifacts {
		full := filepath.Join(dir, filepath.FromSlash(art.Path))
		data, err := os.ReadFile(full)
		if err != nil {
			if os.IsNotExist(err) {
				report.Missing = append(report.Missing, art)
				continue
			}
			report.Unreadable = append(report.Unreadable, art)
			continue
		}
		if HashBytes(data) != art.SHA256 {
			report.Modified = append(report.Modified, art)
			continue
		}
		report.Clean = append(report.Clean, art)
	}
	return report
}

// RemoveEmptyDirs removes empty directories walking up from relDir under root.
func RemoveEmptyDirs(root, relDir string) error {
	root = filepath.Clean(root)
	cur := filepath.Clean(filepath.Join(root, filepath.FromSlash(relDir)))
	for {
		if cur == root || !isWithinRoot(root, cur) {
			return nil
		}
		entries, err := os.ReadDir(cur)
		if err != nil {
			if os.IsNotExist(err) {
				parent := filepath.Dir(cur)
				if parent == cur {
					return nil
				}
				cur = parent
				continue
			}
			return err
		}
		if len(entries) > 0 {
			return nil
		}
		if err := os.Remove(cur); err != nil {
			return err
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			return nil
		}
		cur = parent
	}
}

func isWithinRoot(root, cur string) bool {
	rel, err := filepath.Rel(root, cur)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// RemoveEmptyParents walks up from each artifact path and removes empty dirs under root.
func RemoveEmptyParents(root string, artifactPaths []string) error {
	seen := make(map[string]struct{})
	for _, rel := range artifactPaths {
		dir := filepath.Dir(filepath.FromSlash(rel))
		for dir != "." && dir != "" {
			if _, ok := seen[dir]; ok {
				break
			}
			seen[dir] = struct{}{}
			if err := RemoveEmptyDirs(root, dir); err != nil {
				return err
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}
	return nil
}
