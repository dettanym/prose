package envoyfilter

// TODO: The tracing unit should use these as well.
//  So these constants are shared between the data and control planes.
const (
	PROSE_PRESIDIO_ERROR = "prose_presidio_error"
	PROSE_PII_TYPES      = "prose_pii_types"
	PROSE_OPA_ENFORCE    = "prose_opa_enforce"
	PROSE_OPA_ERROR      = "prose_opa_error"
	PROSE_OPA_DECISION   = "prose_opa_decision"
	PROSE_VIOLATION_TYPE = "prose_violation_type"
)

const (
	DataSharing          string = "DATA_SHARING"
	PurposeOfUseDirect          = "PURPOSE_OF_USE_DIRECT"
	PurposeOfUseIndirect        = "PURPOSE_OF_USE_INDIRECT"
)
