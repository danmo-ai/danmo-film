package port

type FilmAsset struct {
	ID             string         `json:"id"`
	Kind           string         `json:"kind"`
	MimeType       string         `json:"mime_type"`
	FilePath       string         `json:"file_path"`
	ThumbnailPath  string         `json:"thumbnail_path,omitempty"`
	SourceProvider string         `json:"source_provider,omitempty"`
	ProviderRef    string         `json:"provider_ref,omitempty"`
	SourceTaskID   string         `json:"source_task_id,omitempty"`
	Metadata       map[string]any `json:"metadata"`
	CreatedAt      string         `json:"created_at"`
	ThumbnailURL   string         `json:"thumbnail_url"`
	FileURL        string         `json:"file_url"`
}

type CreateAssetOpts struct {
	Kind           string
	MimeType       string
	SourceProvider string
	ProviderRef    string
	SourceTaskID   string
	Metadata       map[string]any
}

type AssetStore interface {
	CreateFromFile(src string, opts CreateAssetOpts) (*FilmAsset, error)
	Get(id string) (*FilmAsset, error)
	ResolvePath(id string) (string, error)
	List(kind string, limit, offset int) ([]*FilmAsset, error)
	FindByProviderRef(providerID, ref string) (*FilmAsset, error)
	Delete(id string) (bool, error)
}

type ProjectSummary struct {
	ID            string `json:"id"`
	Title         string `json:"title"`
	ShotCount     int    `json:"shot_count"`
	KeyframeCount int    `json:"keyframe_count"`
	SegmentCount  int    `json:"segment_count"`
	HasFinal      bool   `json:"has_final"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
}

type ProjectDetail struct {
	ID        string         `json:"id"`
	Title     string         `json:"title"`
	State     map[string]any `json:"state"`
	CreatedAt string         `json:"created_at"`
	UpdatedAt string         `json:"updated_at"`
}

type ProjectStore interface {
	List(limit int) ([]ProjectSummary, error)
	Get(id string) (*ProjectDetail, error)
	Create(title string, state map[string]any) (*ProjectDetail, error)
	Update(id string, title *string, state map[string]any) (*ProjectDetail, error)
	Delete(id string) (bool, error)
}

type SeriesSummary struct {
	ID           string `json:"id"`
	Title        string `json:"title"`
	EpisodeCount int    `json:"episode_count"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

type SeriesDetail struct {
	ID        string         `json:"id"`
	Title     string         `json:"title"`
	State     map[string]any `json:"state"`
	CreatedAt string         `json:"created_at"`
	UpdatedAt string         `json:"updated_at"`
}

type SeriesEpisodeSummary struct {
	SeriesID         string `json:"series_id"`
	EpisodeProjectID string `json:"episode_project_id"`
	EpisodeNumber    int    `json:"episode_number"`
	Title            string `json:"title"`
	ShotCount        int    `json:"shot_count"`
	KeyframeCount    int    `json:"keyframe_count"`
	SegmentCount     int    `json:"segment_count"`
	HasFinal         bool   `json:"has_final"`
	CreatedAt        string `json:"created_at"`
	UpdatedAt        string `json:"updated_at"`
}

type SeriesStore interface {
	List(limit int) ([]SeriesSummary, error)
	Get(id string) (*SeriesDetail, error)
	Create(title string, state map[string]any) (*SeriesDetail, error)
	Update(id string, title *string, state map[string]any) (*SeriesDetail, error)
	Delete(id string) (bool, error)
	ListEpisodes(seriesID string) ([]SeriesEpisodeSummary, error)
	AddEpisode(seriesID, projectID, title string, episodeNumber int) (*SeriesEpisodeSummary, error)
	UpdateEpisodeTitle(seriesID, projectID, title string) (*SeriesEpisodeSummary, error)
	RemoveEpisode(seriesID, projectID string) (bool, error)
	FindSeriesByEpisode(projectID string) (*SeriesDetail, *SeriesEpisodeSummary, error)
	NextEpisodeNumber(seriesID string) (int, error)
	AssetReferenced(assetID string) (bool, string, error)
	LibraryReferenced(entityID string) (bool, string, error)
}

type LibraryEntityVersion struct {
	EntityID   string         `json:"entity_id"`
	Version    int            `json:"version"`
	Kind       string         `json:"kind"`
	Definition map[string]any `json:"definition"`
	CreatedAt  string         `json:"created_at"`
}

type CharacterLibraryStore interface {
	List(limit int) ([]LibraryEntityVersion, error)
	Get(entityID string, version int) (*LibraryEntityVersion, error)
	Create(definition map[string]any) (*LibraryEntityVersion, error)
	NewVersion(entityID string, definition map[string]any) (*LibraryEntityVersion, error)
	ListVersions(entityID string) ([]LibraryEntityVersion, error)
	Delete(entityID string) (bool, error)
}

type SceneLibraryStore interface {
	List(limit int) ([]LibraryEntityVersion, error)
	Get(entityID string, version int) (*LibraryEntityVersion, error)
	Create(definition map[string]any) (*LibraryEntityVersion, error)
	NewVersion(entityID string, definition map[string]any) (*LibraryEntityVersion, error)
	ListVersions(entityID string) ([]LibraryEntityVersion, error)
	Delete(entityID string) (bool, error)
}

type ActivityEvent struct {
	ID         string         `json:"id"`
	ProjectID  string         `json:"project_id"`
	Category   string         `json:"category"`
	EventType  string         `json:"event_type"`
	Phase      string         `json:"phase"`
	Status     string         `json:"status"`
	Summary    string         `json:"summary"`
	TaskID     any            `json:"task_id"`
	ParseRunID any            `json:"parse_run_id"`
	ShotID     string         `json:"shot_id"`
	Detail     map[string]any `json:"detail"`
	CreatedAt  string         `json:"created_at"`
}

type ActivityFilter struct {
	Limit      int
	Offset     int
	Category   string
	Phase      string
	EventType  string
	ParseRunID string
	TaskID     string
	ShotID     string
}

type ActivityStore interface {
	Append(ev ActivityEvent) (*ActivityEvent, error)
	List(projectID string, f ActivityFilter) ([]ActivityEvent, error)
	Count(projectID string, f ActivityFilter) (int, error)
	GetParseRun(projectID, parseRunID string) (map[string]any, error)
}
