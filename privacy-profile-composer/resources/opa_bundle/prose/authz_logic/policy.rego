    package prose.authz_logic

    import rego.v1

    import input.purpose_of_use
    import input.data_items
    import input.external_domain

    default allow := false

    allow if {
        purpose_is_valid
        purpose_is_allowed
        processing_is_allowed
    }

    given_purpose_of_use := input.purpose_of_use
    given_pii_types := input.data_items
    external_domain := input.external_domain

    purpose_is_valid if valid_purposes[given_purpose_of_use]

    valid_purposes contains given_purpose_of_use if {
        given_purpose_of_use in purposes_of_use_set
    }

    purpose_is_allowed if allowed_purposes[given_purpose_of_use]

    allowed_purposes contains purpose if {
        some index, _ in target_policy
        purpose := index
    }

    #Get the list of allowed processing for that purpose of use.
    allowed_processing contains processing if {
        purpose_is_allowed
        processing := target_policy[given_purpose_of_use]
    }

    processing_is_allowed if {
        allowed_processing := target_policy[given_purpose_of_use]
        #For each item in that list,
        every pii_type in given_pii_types {
            some allowed in allowed_processing

            allowed_pii_type := pii_type == allowed.data_item
            valid_pii_type := pii_type in data_items_set
            valid_external_domain := external_domain in allowed.third_parties

            print("Allowed processing for your purpose of use:", [allowed_processing])
            print("Your values: should all be true. IN REALITY: ", allowed_pii_type, valid_pii_type, valid_external_domain)

            allowed_pii_type
            valid_pii_type
            valid_external_domain
        }
    }
