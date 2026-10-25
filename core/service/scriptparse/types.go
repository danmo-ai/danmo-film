package scriptparse

// ScriptPipeline is the single source of truth for multi-stage script breakdown.
type ScriptPipeline struct {
	SchemaVersion  string                    `json:"schema_version"`
	ScriptID       string                    `json:"script_id"`
	VersionID      string                    `json:"version_id"`
	SourceHash     string                    `json:"source_hash"`
	Locale         string                    `json:"locale,omitempty"`
	Title          string                    `json:"title,omitempty"`
	SourceDocument *SourceDocument           `json:"source_document,omitempty"`
	SceneIndex     []SceneIndexEntry         `json:"scene_index,omitempty"`
	StoryMap       *StoryMap                 `json:"story_map,omitempty"`
	EntityRegistry *EntityRegistry           `json:"entity_registry,omitempty"`
	SceneArtifacts map[string]*SceneArtifact `json:"scene_artifacts,omitempty"`
	Continuity     *ContinuityReport         `json:"continuity_report,omitempty"`
	ShotList       []ShotListItem            `json:"shot_list,omitempty"`
	GenPackages    []GenerationPackage       `json:"generation_packages,omitempty"`
	AuditReport    *AuditReport              `json:"audit_report,omitempty"`
	RuleFindings   []RuleFinding             `json:"rule_findings,omitempty"`
	Workflow       WorkflowState             `json:"workflow"`
	SceneChangeSet []SceneChange             `json:"scene_change_set,omitempty"`
	ParseRunID     string                    `json:"parse_run_id,omitempty"`
	LLMCalls       int                       `json:"llm_calls,omitempty"`
	Phases         []map[string]any          `json:"parse_phases,omitempty"`
}

// RuleFinding is a deterministic lint result (separate from LLM continuity/audit issues).
type RuleFinding struct {
	FindingID         string `json:"finding_id"`
	Source            string `json:"source"`   // rule
	Severity          string `json:"severity"` // blocker|major|minor|info
	Action            string `json:"action"`   // fill_field|fix_value|add_element|remove|clamp_ref
	ArtifactID        string `json:"artifact_id,omitempty"`
	SceneID           string `json:"scene_id,omitempty"`
	FieldPath         string `json:"field_path,omitempty"`
	Problem           string `json:"problem"`
	RepairInstruction string `json:"repair_instruction,omitempty"`
	EvidenceQuote     string `json:"evidence_quote,omitempty"`
	SourceStartLine   int    `json:"source_start_line,omitempty"`
	SourceEndLine     int    `json:"source_end_line,omitempty"`
	SuggestedValue    any    `json:"suggested_value,omitempty"`
	NeedsLLM          bool   `json:"needs_llm"`
	Status            string `json:"status,omitempty"` // open|resolved
}

// Repair action constants
const (
	ActionFillField  = "fill_field"
	ActionFixValue   = "fix_value"
	ActionAddElement = "add_element"
	ActionRemove     = "remove"
	ActionClampRef   = "clamp_ref"
)

type SourceDocument struct {
	Text         string   `json:"text"`
	NumberedText string   `json:"numbered_text"`
	Lines        []string `json:"lines"`
	UploadedAt   string   `json:"uploaded_at,omitempty"`
}

type SceneIndexEntry struct {
	SceneID           string `json:"scene_id"`
	SourceSceneNumber string `json:"source_scene_number,omitempty"`
	Heading           string `json:"heading"`
	IntExt            string `json:"int_ext,omitempty"`
	LocationText      string `json:"location_text,omitempty"`
	StoryTime         string `json:"story_time,omitempty"`
	SourceStartLine   int    `json:"source_start_line"`
	SourceEndLine     int    `json:"source_end_line"`
	SourceHash        string `json:"source_hash"`
	Synopsis          string `json:"synopsis,omitempty"`
	Text              string `json:"text,omitempty"`
}

type StoryMap struct {
	Genre         string              `json:"genre,omitempty"`
	Logline       string              `json:"logline"`
	Mood          string              `json:"mood,omitempty"`
	StyleAnchor   string              `json:"style_anchor,omitempty"`
	CoreConflict  string              `json:"core_conflict,omitempty"`
	Characters    []RegistryCharacter `json:"characters,omitempty"`
	Locations     []RegistryLocation  `json:"locations,omitempty"`
	Hypotheses    []string            `json:"hypotheses,omitempty"`
	OpenQuestions []string            `json:"open_questions,omitempty"`
	ReviewStatus  string              `json:"review_status,omitempty"`
}

type EntityRegistry struct {
	Characters []RegistryCharacter `json:"characters,omitempty"`
	Locations  []RegistryLocation  `json:"locations,omitempty"`
	Props      []RegistryProp      `json:"props,omitempty"`
}

type RegistryCharacter struct {
	CharacterID        string   `json:"character_id"`
	Name               string   `json:"name"`
	Aliases            []string `json:"aliases,omitempty"`
	Role               string   `json:"role,omitempty"` // protagonist|supporting|extra
	Basis              string   `json:"basis,omitempty"`
	AgeRange           string   `json:"age_range,omitempty"`
	GenderPresentation string   `json:"gender_presentation,omitempty"`
	IdentityBaseline   string   `json:"identity_baseline,omitempty"`
	VisualIdentity     string   `json:"visual_identity,omitempty"`
	VoiceManner        string   `json:"voice_manner,omitempty"`
	PerformanceNotes   string   `json:"performance_notes,omitempty"`
	LookBody           string   `json:"look_body,omitempty"` // wardrobe / hair / accessories only
	ReviewStatus       string   `json:"review_status,omitempty"`
}

type RegistryLocation struct {
	LocationID   string `json:"location_id"`
	Name         string `json:"name"`
	Body         string `json:"body,omitempty"`
	ReviewStatus string `json:"review_status,omitempty"`
}

type RegistryProp struct {
	ElementID    string   `json:"element_id"`
	Name         string   `json:"name"`
	Category     string   `json:"category,omitempty"` // prop|set_dressing|wardrobe|sfx|sound
	Basis        string   `json:"basis,omitempty"`
	SceneIDs     []string `json:"scene_ids,omitempty"`
	ReviewStatus string   `json:"review_status,omitempty"`
}

type Evidence struct {
	ScriptVersion   string `json:"script_version,omitempty"`
	SceneID         string `json:"scene_id,omitempty"`
	SourceStartLine int    `json:"source_start_line,omitempty"`
	SourceEndLine   int    `json:"source_end_line,omitempty"`
	Quote           string `json:"quote,omitempty"`
}

type BreakdownElement struct {
	ElementID        string    `json:"element_id"`
	Category         string    `json:"category"`
	Name             string    `json:"name"`
	Basis            string    `json:"basis"` // explicit|implied|proposed|unknown
	Evidence         *Evidence `json:"evidence,omitempty"`
	ContinuityStates []string  `json:"continuity_states,omitempty"`
	ReviewStatus     string    `json:"review_status,omitempty"`
}

type SceneBreakdown struct {
	ArtifactID          string             `json:"artifact_id"`
	SceneID             string             `json:"scene_id"`
	Heading             string             `json:"heading,omitempty"`
	IntExt              string             `json:"int_ext,omitempty"`
	StoryTime           string             `json:"story_time,omitempty"`
	CastPresent         []string           `json:"cast_present,omitempty"`
	MentionedNotPresent []string           `json:"mentioned_not_present,omitempty"`
	Elements            []BreakdownElement `json:"elements,omitempty"`
	ReviewQuestions     []string           `json:"review_questions,omitempty"`
	ReviewStatus        string             `json:"review_status,omitempty"`
}

type SceneBeat struct {
	BeatID          string `json:"beat_id"`
	SourceEvent     string `json:"source_event"`
	BeatType        string `json:"beat_type,omitempty"`
	Effect          string `json:"effect,omitempty"`
	SourceStartLine int    `json:"source_start_line,omitempty"`
	SourceEndLine   int    `json:"source_end_line,omitempty"`
	Interpretive    bool   `json:"interpretive,omitempty"`
}

type SceneAnalysis struct {
	ArtifactID       string      `json:"artifact_id"`
	SceneID          string      `json:"scene_id"`
	DramaticFunction string      `json:"dramatic_function,omitempty"`
	Beats            []SceneBeat `json:"beats,omitempty"`
	EntryState       string      `json:"entry_state,omitempty"`
	ExitState        string      `json:"exit_state,omitempty"`
	Deletability     string      `json:"deletability,omitempty"`
	Confidence       string      `json:"confidence,omitempty"`
	OpenQuestions    []string    `json:"open_questions,omitempty"`
	ReviewStatus     string      `json:"review_status,omitempty"`
}

type SceneArtifact struct {
	SceneID      string          `json:"scene_id"`
	Breakdown    *SceneBreakdown `json:"breakdown,omitempty"`
	Analysis     *SceneAnalysis  `json:"analysis,omitempty"`
	ReviewStatus string          `json:"review_status,omitempty"`
}

type ContinuityIssue struct {
	IssueID    string `json:"issue_id"`
	Severity   string `json:"severity"` // blocker|major|minor|info
	Action     string `json:"action,omitempty"`
	SceneID    string `json:"scene_id,omitempty"`
	FieldPath  string `json:"field_path,omitempty"`
	Problem    string `json:"problem"`
	Evidence   string `json:"evidence_quote,omitempty"`
	RepairHint string `json:"repair_instruction,omitempty"`
	Status     string `json:"status,omitempty"`
}

type ContinuityReport struct {
	ArtifactID       string              `json:"artifact_id"`
	MergeSuggestions []string            `json:"merge_suggestions,omitempty"`
	Issues           []ContinuityIssue   `json:"issues,omitempty"`
	CastMatrix       map[string][]string `json:"cast_matrix,omitempty"` // character_id -> scene_ids
	PropMatrix       map[string][]string `json:"prop_matrix,omitempty"`
	ReviewStatus     string              `json:"review_status,omitempty"`
}

type ShotListItem struct {
	ShotID             string   `json:"shot_id"`
	SceneID            string   `json:"scene_id"`
	SourceBeatIDs      []string `json:"source_beat_ids,omitempty"`
	ShotSize           string   `json:"shot_size,omitempty"`
	CameraAngle        string   `json:"camera_angle,omitempty"`
	CameraMovement     string   `json:"camera_movement,omitempty"`
	VisualAction       string   `json:"visual_action,omitempty"`
	Dialogue           string   `json:"dialogue,omitempty"`
	DurationSec        float64  `json:"duration_sec,omitempty"`
	ContinuityNotes    string   `json:"continuity_notes,omitempty"`
	Basis              string   `json:"basis,omitempty"` // proposed
	CharactersOnScreen []string `json:"characters_on_screen,omitempty"`
	ReviewStatus       string   `json:"review_status,omitempty"`
}

type GenerationPackage struct {
	PackageID      string   `json:"package_id"`
	ShotID         string   `json:"shot_id"`
	SceneID        string   `json:"scene_id,omitempty"`
	KeyframePrompt string   `json:"keyframe_prompt,omitempty"`
	VideoPrompt    string   `json:"video_prompt,omitempty"`
	NegativePrompt string   `json:"negative_prompt,omitempty"`
	StartVisual    string   `json:"start_visual,omitempty"`
	EndVisual      string   `json:"end_visual,omitempty"`
	CharacterIDs   []string `json:"character_ids,omitempty"`
	LocationID     string   `json:"location_id,omitempty"`
	PropIDs        []string `json:"prop_ids,omitempty"`
	ReviewItems    []string `json:"review_items,omitempty"`
	ReviewStatus   string   `json:"review_status,omitempty"`
}

type AuditIssue struct {
	IssueID           string `json:"issue_id"`
	Severity          string `json:"severity"`
	Action            string `json:"action,omitempty"`
	ArtifactID        string `json:"artifact_id,omitempty"`
	FieldPath         string `json:"field_path,omitempty"`
	Problem           string `json:"problem"`
	EvidenceQuote     string `json:"evidence_quote,omitempty"`
	RepairInstruction string `json:"repair_instruction,omitempty"`
	Status            string `json:"status,omitempty"`
}

type AuditReport struct {
	ArtifactID   string       `json:"artifact_id"`
	Issues       []AuditIssue `json:"issues,omitempty"`
	ReviewStatus string       `json:"review_status,omitempty"`
}

type WorkflowState struct {
	Stage          string `json:"stage"`                     // r0|r1|r2|r3|r4|gate1|r5|r6|r7|gate2|completed
	Status         string `json:"status"`                    // queued|running|awaiting_review|repairing|failed|completed
	AwaitingReview string `json:"awaiting_review,omitempty"` // facts|shots
	FactsApproved  bool   `json:"facts_approved"`
	ShotsApproved  bool   `json:"shots_approved"`
	Error          string `json:"error,omitempty"`
}

type SceneChange struct {
	SceneID string `json:"scene_id"`
	Status  string `json:"status"` // unchanged|modified|added|removed|moved
}

// Review statuses
const (
	ReviewPending       = "pending"
	ReviewApproved      = "approved"
	ReviewRejected      = "rejected"
	ReviewNeedsRevision = "needs_revision"
	ReviewObsolete      = "obsolete"
)

// Workflow stages
const (
	StageR0    = "r0"
	StageR1    = "r1"
	StageR2    = "r2"
	StageR3    = "r3"
	StageR4    = "r4"
	StageGate1 = "gate1"
	StageR5    = "r5"
	StageR6    = "r6"
	StageR7    = "r7"
	StageGate2 = "gate2"
	StageDone  = "completed"
)

const PipelineSchemaVersion = "1.0"
