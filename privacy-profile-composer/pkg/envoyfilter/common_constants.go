package envoyfilter

// TODO: The tracing unit should use these as well.
//  So these constants are shared between the data and control planes.

const (
	PROSE_SIDECAR_DIRECTION = "prose_sidecar_direction"
	PROSE_DATA_FLOW         = "prose_data_flow"
	PROSE_PII_TYPES         = "prose_pii_types"
	PROSE_OPA_ENFORCE       = "prose_opa_enforce"
	PROSE_OPA_DECISION      = "prose_opa_decision"
	PROSE_VIOLATION_TYPE    = "prose_violation_type"
	PROSE_EXTERNAL_DOMAIN   = "prose_external_domain"
	PROSE_PRESIDIO_ERROR    = "prose_presidio_error"
	PROSE_JSON_BODY_ERROR   = "prose_json_body_error"
	PROSE_OPA_ERROR         = "prose_opa_error"
)

const (
	DataSharing          string = "DATA_SHARING"
	PurposeOfUseDirect          = "PURPOSE_OF_USE_DIRECT"
	PurposeOfUseIndirect        = "PURPOSE_OF_USE_INDIRECT"
)
