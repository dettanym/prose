package envoyfilter

// TODO: The tracing unit should use these as well.
const (
	PROSE_PRESIDIO_ERROR = "prose_presidio_error"
	PROSE_PII_TYPES      = "prose_pii_types"
	PROSE_OPA_STATUS     = "prose_opa_status"
	PROSE_OPA_ERROR      = "prose_opa_error"
	PROSE_OPA_DECISION   = "prose_opa_decision"
	PROSE_VIOLATION_TYPE = "prose_violation_type"
)

const (
	DataSharing          string = "DATA_SHARING"
	PurposeOfUseDirect          = "PURPOSE_OF_USE_DIRECT"
	PurposeOfUseIndirect        = "PURPOSE_OF_USE_INDIRECT"
)