    package prose
    import future.keywords

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

    purposes_of_use_set := {
         "advertising",
         "authentication",
         "shipping",
         "payment"
    }

    data_items_set := {
        "CREDIT_CARD",
        "NRP",
        "US_ITIN",
        "PERSON",
        "US_BANK_NUMBER",
        "US_PASSPORT",
        "IP_ADDRESS",
        "US_DRIVER_LICENSE",
        "CRYPTO",
        "URL",
        "PHONE_NUMBER",
        "IBAN_CODE",
        "DATE_TIME",
        "LOCATION",
        "EMAIL_ADDRESS",
        "US_SSN",
    }

    target_policy = {
        "advertising": [
            {"data_item": "DATE_TIME", "third_parties": ["google.com"]},
            {"data_item": "LOCATION", "third_parties": []}
        ],
        "authentication": [
            {"data_item": "EMAIL_ADDRESS", "third_parties": []},
            {"data_item": "PERSON", "third_parties": []},
            {"data_item": "IP_ADDRESS", "third_parties": []}
        ],
        "shipping": [
            {"data_item": "PERSON", "third_parties": []},
            {"data_item": "LOCATION", "third_parties": ["canadapost-postescanada.ca"] },
            {"data_item": "EMAIL_ADDRESS", "third_parties": ["canadapost-postescanada.ca"]},
            {"data_item": "PHONE_NUMBER", "third_parties": ["canadapost-postescanada.ca"]}
        ],
        "payment": [
            {"data_item": "PERSON", "third_parties": []},
        ]
    }

