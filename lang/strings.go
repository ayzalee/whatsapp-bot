package lang

type Strings struct {
	Pong        string
	PongLatency string

	MetaUsage string

	GroupOnly string
	SudoOnly  string

	SetPrefixUsage   string
	SetPrefixUpdated string
	SaveFailed       string

	SetSudoUsage  string
	DelSudoUsage  string
	SudoAdded     string
	SudoRemoved   string
	SudoNotFound  string
	SudoList      string
	SudoListEmpty string
	UnknownAction string

	SetModeUsage   string
	ModePublicSet  string
	ModePrivateSet string

	EnableCmdUsage  string
	DisableCmdUsage string
	CmdEnabled      string
	CmdDisabledOK   string
	CmdIsDisabled   string
	CmdNotFound     string

	BanUsage      string
	DelBanUsage   string
	UserBanned    string
	UserUnbanned  string
	UserNotBanned string
	BanList       string
	BanListEmpty  string

	GCDisabledSet     string
	GCEnabledSet      string
	GCAlreadyDisabled string
	GCAlreadyEnabled  string

	LangCurrent string
	LangSet     string
	LangUnknown string
	LangUsage   string

	MenuGreeting string

	UserNotFound    string
	GroupInfoFailed string
	UserResolveFail string
	BotNotAdmin     string
	SenderNotAdmin  string

	PromoteUsage        string
	PromoteAlreadyAdmin string
	PromoteOK           string
	DemoteUsage         string
	DemoteNotAdmin      string
	DemoteSuperAdmin    string
	DemoteOK            string

	KickUsage      string
	KickSuperAdmin string
	KickOK         string
	KickAllStart   string
	KickAllDone    string

	MuteUsage      string
	MuteAlready    string
	MuteOK         string
	UnmuteUsage    string
	UnmuteNotMuted string
	UnmuteOK       string

	MessagesEmpty  string
	MessagesHeader string
	ActiveHeader   string
	ActiveEmpty    string
	InactiveHeader string
	InactiveEmpty  string

	WarnUsage   string
	WarnText    string
	WarnKicked  string
	WarnBlocked string

	AntilinkUsage      string
	AntilinkStatus     string
	AntilinkOff        string
	AntilinkOn         string
	AntilinkSetUsage   string
	AntilinkUnknownAct string
	AntilinkNotify     string
	AntilinkSet        string

	AntiwordUsage       string
	AntiwordEmpty       string
	AntiwordList        string
	AntiwordAddUsage    string
	AntiwordAdded       string
	AntiwordRemoveUsage string
	AntiwordRemoved     string

	AFKEnabled    string
	AFKOff        string
	AFKNotActive  string
	AFKSetUsage   string
	AFKDefaultMsg string
	AFKAutoReply  string

	AntispamUsage   string
	AntispamStatus  string
	AntispamOn      string
	AntispamOff     string
	AntispamAllowed string
	AntispamWarn    string

	ShhUsage    string
	ShhAlready  string
	ShhOK       string
	ShhOffUsage string
	ShhNotShhed string
	ShhOffOK    string

	BlockUsage   string
	BlockOK      string
	UnblockUsage string
	UnblockOK    string

	NewGCUsage        string
	NewGCNameTooLong  string
	NewGCCreating     string
	NewGCSettingDesc  string
	NewGCFetchingIcon string
	NewGCFetchingLink string
	NewGCFailed       string
	NewGCDone         string
	NewGCDefaultDesc  string

	FilterNone     string
	FilterList     string
	FilterSetUsage string
	FilterSet      string
	FilterDelUsage string
	FilterDeleted  string
	FilterNotFound string

	AntistatusOn     string
	AntistatusOff    string
	AntistatusNotify string

	AntcallAlreadyOn  string
	AntcallOn         string
	AntcallAlreadyOff string
	AntcallOff        string
	AntcallStatus     string

	OnlineAlready  string
	OnlineOn       string
	OfflineAlready string
	OnlineOff      string
	OnlineStatus   string

	StatusEnabled    string
	StatusSkip       string
	StatusOnly       string
	StatusDisabled   string
	StatusNoDL       string
	StatusReset      string
	StatusFwdTo      string
	StatusAlreadyOn  string
	StatusOn         string
	StatusAlreadyOff string
	StatusOff        string
	StatusInfo       string

	ReadAlreadyOn  string
	ReadOn         string
	ReadAlreadyOff string
	ReadOff        string
	ReadStatus     string

	AntiDelAlreadyOn string
	AntiDelOn        string
	AntiDelOff       string
	AntiDelStatus    string

	StickerNoReply string
	StickerFailed  string

	SetVarUsage   string
	SetVarInvalid string
	SetVarOK      string
	GetVarFailed  string
	DelVarUsage   string
	DelVarOK      string

	VVUsage       string
	VVFailed      string
	VVUnsupported string

	WhoisNotFound string
	WhoisFailed   string
	WhoisCaption  string

	InviteLink   string
	InviteFailed string

	MediaNotFound string
	DlUsage       string
	DlFailed      string
	DlNoFile      string
	IgUsage       string

	DelUsage string

	PinOK       string
	PinFailed   string
	UnpinOK     string
	UnpinFailed string

	MsgPinOK       string
	MsgPinFailed   string
	MsgUnpinOK     string
	MsgUnpinFailed string

	ArchiveOK       string
	ArchiveFailed   string
	UnarchiveOK     string
	UnarchiveFailed string

	StarUsage    string
	StarOK       string
	StarFailed   string
	UnstarUsage  string
	UnstarOK     string
	UnstarFailed string

	LeaveOK string

	ClearOK     string
	ClearFailed string

	ReportUsage    string
	ReportDone     string
	ReportDoneNoID string
	ReportFailed   string

	MediaNoReply    string
	MediaProcessing string
	MediaFailed     string
	TrimUsage       string

	PluginRemoveUsage string
	PluginNotFound    string
	PluginRemoveFail  string
	PluginRemoved     string
	PluginManager     string
	PluginFetching    string
	PluginFetchFail   string
	PluginURLFail     string
	PluginRejected    string
	PluginSaveFail    string
	PluginSaved       string
	PluginBuildFail   string
	PluginBinaryFail  string
	PluginDone        string
	PluginDirFail     string
	PluginNone        string
	PluginList        string

	ChCreateUsage          string
	ChCreating             string
	ChCreateOK             string
	ChCreateFailed         string
	AntiModOn              string
	AntiModOff             string
	AntiModStatus          string
	AntiModDemote          string
	AntiModPromote         string
	PdmOn                  string
	PdmOff                 string
	PdmStatus              string
	CaptionUsage           string
	CaptionUnsupported     string
	ForwardUsage           string
	ForwardNoTarget        string
	ForwardDone            string
	ForwardFailed          string
	CompressUsage          string
	CompressReplyVideo     string
	CompressDownloadFailed string
	CompressProgress       string
	CompressFailed         string
	CompressTempFailed     string
	CompressTempWriteFail  string
	CompressOpenFailed     string
	CompressUploadFailed   string
	DlDownloading          string
	DlTempDirFailed        string
	DlStatFailed           string
	DlOpenFailed           string
	DlDocUploadFailed      string
	DlAudioUploadFailed    string
	DlVideoUploadFailed    string
	GreetEnabled           string
	GreetDisabled          string
	GreetStatus            string
	GreetSetUsage          string
	GreetSetOK             string
	GreetUsage             string
	AntipornWarnText       string
	AntipornOn             string
	AntipornOff            string
	AntipornStatus         string
	AntipornNoCreds        string
	UploadUsage            string
	UploadFetchFailed      string
	UploadHTTPFailed       string
	UploadReadFailed       string
	UploadFailed           string
	TgUsage                string
	TgStopMsg              string
	TgNoToken              string
	TgNoPackName           string
	TgFetchFailed          string
	TgReadFailed           string
	TgInvalidPack          string
	TgEmptyPack            string
	TgFoundSending         string
	TgResultLine           string
	TgSkippedLine          string
	TgStoppedLine          string
	JoinUsage              string
	JoinNoCode             string
	JoinFailed             string
	JoinOK                 string
}
