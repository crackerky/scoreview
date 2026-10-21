package httpapi

// このファイルは閲覧用エンドポイントの応答 DTO。形は docs/api.md が正本で、キーは camelCase。
// 値がないことは null で表し、キーを省略しない(nullable はポインタ)。配列は空でも [] で返す。
// 例外は対戦表のセル(matrixCellDTO)で、api.md の union 型どおり match 以外は kind だけを持つ。

type i18nDTO struct {
	Ja string `json:"ja"`
	En string `json:"en"`
}

type frameDTO struct {
	ID           int64              `json:"id"`
	Name         i18nDTO            `json:"name"`
	ShortName    string             `json:"shortName"`
	Labels       labelsDTO          `json:"labels"`
	ShowSchedule bool               `json:"showSchedule"`
	Divisions    []frameDivisionDTO `json:"divisions"`
	Galleries    []frameGalleryDTO  `json:"galleries"`
	Banners      []bannerDTO        `json:"banners"`
}

type labelsDTO struct {
	TeamList string `json:"teamList"`
	Groups   string `json:"groups"`
}

type frameDivisionDTO struct {
	ID              int64           `json:"id"`
	Name            i18nDTO         `json:"name"`
	ScheduleEnabled bool            `json:"scheduleEnabled"`
	MenuSuffix      string          `json:"menuSuffix"`
	Groups          []frameGroupDTO `json:"groups"`
}

type frameGroupDTO struct {
	ID     int64   `json:"id"`
	Name   i18nDTO `json:"name"`
	Linked bool    `json:"linked"`
}

type frameGalleryDTO struct {
	ID   int64   `json:"id"`
	Name i18nDTO `json:"name"`
}

type bannerDTO struct {
	ImageURL string  `json:"imageUrl"`
	LinkURL  *string `json:"linkUrl"`
}

type idI18nDTO struct {
	ID   int64   `json:"id"`
	Name i18nDTO `json:"name"`
}

type teamRefDTO = idI18nDTO

type playerRefDTO struct {
	ID     int64   `json:"id"`
	Number string  `json:"number"`
	Name   i18nDTO `json:"name"`
	Gender *string `json:"gender"`
}

type placeholderDTO struct {
	Code    string `json:"code"`
	Outcome string `json:"outcome"`
}

type matchSideDTO struct {
	Team        *teamRefDTO     `json:"team"`
	Placeholder *placeholderDTO `json:"placeholder"`
	Score       *int            `json:"score"`
	LabelEn     string          `json:"labelEn"`
}

type matchRowDTO struct {
	MatchID   int64        `json:"matchId"`
	HasDetail bool         `json:"hasDetail"`
	Code      string       `json:"code"`
	Date      *string      `json:"date"`
	Time      *string      `json:"time"`
	Field     *string      `json:"field"`
	Status    string       `json:"status"`
	Left      matchSideDTO `json:"left"`
	Right     matchSideDTO `json:"right"`
}

type tallyDTO struct {
	Games        int `json:"games"`
	Wins         int `json:"wins"`
	Draws        int `json:"draws"`
	Losses       int `json:"losses"`
	GoalsFor     int `json:"goalsFor"`
	GoalsAgainst int `json:"goalsAgainst"`
}

// GET /api/v1/tournaments

type tournamentListDTO struct {
	Sections []tournamentSectionDTO `json:"sections"`
}

type tournamentSectionDTO struct {
	Status      string      `json:"status"`
	Tournaments []idI18nDTO `json:"tournaments"`
}

// GET /api/v1/tournaments/{id}

type tournamentTopDTO struct {
	Frame     frameDTO         `json:"frame"`
	LogoURL   *string          `json:"logoUrl"`
	TopHTML   i18nDTO          `json:"topHtml"`
	Divisions []topDivisionDTO `json:"divisions"`
}

type topDivisionDTO struct {
	ID              int64   `json:"id"`
	Name            i18nDTO `json:"name"`
	ScheduleLinked  bool    `json:"scheduleLinked"`
	StandingsLinked bool    `json:"standingsLinked"`
}

// GET /api/v1/tournaments/{id}/teams

type teamListDTO struct {
	Frame     frameDTO              `json:"frame"`
	Divisions []teamListDivisionDTO `json:"divisions"`
}

type teamListDivisionDTO struct {
	ID        int64             `json:"id"`
	Name      i18nDTO           `json:"name"`
	Kind      string            `json:"kind"`
	EntryKind string            `json:"entryKind"`
	Teams     []teamListTeamDTO `json:"teams"`
}

type teamListTeamDTO struct {
	ID          int64   `json:"id"`
	Name        i18nDTO `json:"name"`
	BarStyle    string  `json:"barStyle"`
	MemberCount *int    `json:"memberCount"`
}

// GET /api/v1/tournaments/{id}/schedule と /api/v1/divisions/{id}/schedule

type scheduleDTO struct {
	Frame    frameDTO           `json:"frame"`
	Scope    string             `json:"scope"`
	Division *idI18nDTO         `json:"division"`
	Filters  scheduleFiltersDTO `json:"filters"`
	Sort     *scheduleSortDTO   `json:"sort"`
	Rows     []scheduleRowDTO   `json:"rows"`
}

type scheduleFiltersDTO struct {
	Dates    []string            `json:"dates"`
	Teams    []scheduleTeamDTO   `json:"teams"`
	Fields   []string            `json:"fields"`
	Selected scheduleSelectedDTO `json:"selected"`
}

type scheduleTeamDTO struct {
	Value string  `json:"value"`
	Name  i18nDTO `json:"name"`
}

type scheduleSelectedDTO struct {
	YMD   string `json:"ymd"`
	Team  string `json:"team"`
	Field string `json:"field"`
}

type scheduleSortDTO struct {
	Key string `json:"key"`
	Dir string `json:"dir"`
}

type scheduleRowDTO struct {
	matchRowDTO
	Division scheduleDivisionDTO `json:"division"`
}

type scheduleDivisionDTO struct {
	ID   int64   `json:"id"`
	Name i18nDTO `json:"name"`
	Kind string  `json:"kind"`
}

// GET /api/v1/divisions/{id}/standings

type standingsDTO struct {
	Frame        frameDTO            `json:"frame"`
	Division     idI18nDTO           `json:"division"`
	Standings    []standingRowDTO    `json:"standings"`
	Leaders      []leaderRowDTO      `json:"leaders"`
	BlockLeaders []blockLeaderRowDTO `json:"blockLeaders"`
}

type standingRowDTO struct {
	tallyDTO
	Team teamRefDTO `json:"team"`
	Rank *int       `json:"rank"`
}

type leaderRowDTO struct {
	Team    teamRefDTO   `json:"team"`
	Player  playerRefDTO `json:"player"`
	Games   int          `json:"games"`
	Assists int          `json:"assists"`
	Goals   int          `json:"goals"`
}

type blockLeaderRowDTO struct {
	Team   teamRefDTO   `json:"team"`
	Player playerRefDTO `json:"player"`
	Games  int          `json:"games"`
	Blocks int          `json:"blocks"`
}

// GET /api/v1/groups/{id}

type groupDTO struct {
	Frame   frameDTO      `json:"frame"`
	Group   groupInfoDTO  `json:"group"`
	Pool    *poolDTO      `json:"pool"`
	Bracket *bracketDTO   `json:"bracket"`
	Image   *imageDTO     `json:"image"`
	Matches []matchRowDTO `json:"matches"`
}

type groupInfoDTO struct {
	ID       int64     `json:"id"`
	Name     string    `json:"name"`
	Kind     string    `json:"kind"`
	Division idI18nDTO `json:"division"`
}

type poolDTO struct {
	Teams   []teamRefDTO      `json:"teams"`
	Matrix  [][]matrixCellDTO `json:"matrix"`
	Results []poolResultDTO   `json:"results"`
}

// matrixCellDTO は api.md の union 型 { kind: 'self'|'lower'|'empty' } | { kind: 'match', ... }。
// match 以外のセルは matchId、status、text のキーを持たない。
type matrixCellDTO struct {
	Kind    string  `json:"kind"`
	MatchID *int64  `json:"matchId,omitempty"`
	Status  *string `json:"status,omitempty"`
	Text    *string `json:"text,omitempty"`
}

type poolResultDTO struct {
	tallyDTO
	Team   teamRefDTO `json:"team"`
	Points int        `json:"points"`
	Rank   *int       `json:"rank"`
}

type bracketDTO struct {
	Pattern string             `json:"pattern"`
	PtnName string             `json:"ptnName"`
	Columns int                `json:"columns"`
	Rows    [][]bracketCellDTO `json:"rows"`
}

type bracketCellDTO struct {
	Borders bordersDTO  `json:"borders"`
	Kind    string      `json:"kind"`
	Text    string      `json:"text"`
	Team    *teamRefDTO `json:"team"`
	MatchID *int64      `json:"matchId"`
	Status  *string     `json:"status"`
}

type bordersDTO struct {
	Top    bool `json:"top"`
	Bottom bool `json:"bottom"`
	Left   bool `json:"left"`
	Right  bool `json:"right"`
}

type imageDTO struct {
	URL    string `json:"url"`
	Width  *int   `json:"width"`
	Height *int   `json:"height"`
}

// GET /api/v1/teams/{id}

type teamPageDTO struct {
	Frame   frameDTO        `json:"frame"`
	Team    teamInfoDTO     `json:"team"`
	Record  teamRecordDTO   `json:"record"`
	Matches []matchRowDTO   `json:"matches"`
	Players []playerLineDTO `json:"players"`
	Staff   []staffLineDTO  `json:"staff"`
	// ShowSotg は SOTG スコア一覧を出すか(行は matches から作る。値は空)
	ShowSotg bool `json:"showSotg"`
}

type teamInfoDTO struct {
	ID        int64   `json:"id"`
	FullName  i18nDTO `json:"fullName"`
	ShortName i18nDTO `json:"shortName"`
	Comment   i18nDTO `json:"comment"`
	Layout    string  `json:"layout"`
}

type teamRecordDTO struct {
	tallyDTO
	Rank *int `json:"rank"`
}

type playerLineDTO struct {
	Player  playerRefDTO `json:"player"`
	Games   int          `json:"games"`
	Assists int          `json:"assists"`
	Goals   int          `json:"goals"`
	Blocks  int          `json:"blocks"`
	Absent  bool         `json:"absent"`
}

type staffLineDTO struct {
	ID     int64   `json:"id"`
	Role   i18nDTO `json:"role"`
	Name   i18nDTO `json:"name"`
	Gender *string `json:"gender"`
}

// GET /api/v1/matches/{id}

type matchPageDTO struct {
	Frame   frameDTO       `json:"frame"`
	Match   matchInfoDTO   `json:"match"`
	Events  []eventLineDTO `json:"events"`
	Lineups lineupsDTO     `json:"lineups"`
	Stats   matchStatsDTO  `json:"stats"`
}

type matchInfoDTO struct {
	ID         int64             `json:"id"`
	Code       string            `json:"code"`
	GroupName  *string           `json:"groupName"`
	Date       *string           `json:"date"`
	Time       *string           `json:"time"`
	Field      *string           `json:"field"`
	Status     string            `json:"status"`
	Left       matchTeamScoreDTO `json:"left"`
	Right      matchTeamScoreDTO `json:"right"`
	Note       string            `json:"note"`
	HideBlocks bool              `json:"hideBlocks"`
	HideStats  bool              `json:"hideStats"`
	Breadcrumb breadcrumbDTO     `json:"breadcrumb"`
}

type matchTeamScoreDTO struct {
	Team     *teamRefDTO `json:"team"`
	FullName *i18nDTO    `json:"fullName"`
	Score    *int        `json:"score"`
}

type breadcrumbDTO struct {
	DivisionID   int64   `json:"divisionId"`
	DivisionName i18nDTO `json:"divisionName"`
	GroupID      *int64  `json:"groupId"`
}

type eventLineDTO struct {
	ElapsedSec int           `json:"elapsedSec"`
	Kind       string        `json:"kind"`
	Side       *string       `json:"side"`
	Scorer     *playerRefDTO `json:"scorer"`
	Assist     *playerRefDTO `json:"assist"`
	Callahan   bool          `json:"callahan"`
	LeftScore  int           `json:"leftScore"`
	RightScore int           `json:"rightScore"`
}

type lineupsDTO struct {
	Left  []lineupLineDTO `json:"left"`
	Right []lineupLineDTO `json:"right"`
}

type lineupLineDTO struct {
	Player  playerRefDTO `json:"player"`
	Assists int          `json:"assists"`
	Goals   int          `json:"goals"`
	Blocks  int          `json:"blocks"`
}

type matchStatsDTO struct {
	Left  sideStatsDTO `json:"left"`
	Right sideStatsDTO `json:"right"`
}

type sideStatsDTO struct {
	Goals        int `json:"goals"`
	FromOffense  int `json:"fromOffense"`
	FromDefense  int `json:"fromDefense"`
	FromTurnover int `json:"fromTurnover"`
	Timeouts     int `json:"timeouts"`
}

// GET /api/v1/players/{id}

type playerPageDTO struct {
	Frame     frameDTO             `json:"frame"`
	Player    playerWithRoleDTO    `json:"player"`
	Team      teamRefDTO           `json:"team"`
	Prev      playerWithRoleDTO    `json:"prev"`
	Next      playerWithRoleDTO    `json:"next"`
	StatsKind string               `json:"statsKind"`
	Stats     playerStatsDTO       `json:"stats"`
	Events    []playerEventLineDTO `json:"events"`
}

type playerWithRoleDTO struct {
	playerRefDTO
	Role *i18nDTO `json:"role"`
}

type playerStatsDTO struct {
	Games     int `json:"games"`
	Assists   int `json:"assists"`
	Goals     int `json:"goals"`
	Callahans int `json:"callahans"`
	Greatests int `json:"greatests"`
	Blocks    int `json:"blocks"`
}

type playerEventLineDTO struct {
	Code       string        `json:"code"`
	Date       *string       `json:"date"`
	Opponent   *teamRefDTO   `json:"opponent"`
	ElapsedSec int           `json:"elapsedSec"`
	Callahan   bool          `json:"callahan"`
	Scorer     *playerRefDTO `json:"scorer"`
	Assist     *playerRefDTO `json:"assist"`
}

// GET /api/v1/galleries/{id}

type galleryPageDTO struct {
	Frame   frameDTO   `json:"frame"`
	Gallery galleryDTO `json:"gallery"`
}

type galleryDTO struct {
	ID     int64      `json:"id"`
	Name   string     `json:"name"`
	Photos []photoDTO `json:"photos"`
}

type photoDTO struct {
	ThumbURL string `json:"thumbUrl"`
	FullURL  string `json:"fullUrl"`
}
