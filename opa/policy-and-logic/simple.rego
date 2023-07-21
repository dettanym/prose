   package authz
    import input.parsed_body
    import future.keywords

    default allow := false

    allow if {
        parsed_path[0] == "health"
        http_request.method == "GET"
    }

    allow if {
        #print(purpose_is_valid, purpose_is_allowed, processing_is_allowed)
        purpose_is_valid
        purpose_is_allowed
        processing_is_allowed
    }

    given_purpose_of_use := input.parsed_body.purpose_of_use

    purpose_is_valid if valid_purposes[given_purpose_of_use]

    valid_purposes contains given_purpose_of_use if {
        given_purpose_of_use in purposes_of_use_set
    }

    purposes_of_use_set := {
         "advertising",
         "authentication",
         "shipping",
         "payment"
    }

    purpose_is_allowed if allowed_purposes[given_purpose_of_use]

    allowed_purposes contains purpose if {
        some index, _ in target_policy
        purpose := index
    }

    #Get the list of allowed processing for that purpose of use.
    allowed_processing contains processing if {
        purpose_is_allowed
        print(given_purpose_of_use)
        processing := target_policy[given_purpose_of_use]
    }

    processing_is_allowed if {
        allowed_processing := target_policy[given_purpose_of_use]
        #For each item in that list,
        every processing in input.parsed_body.processing {
            some allowed in allowed_processing
            processing.data_item in data_items_set
            processing.data_item == allowed.data_item
            every third_party in processing.third_parties {
                third_party in allowed.third_parties
            }
            #processing.third_parties in allowed_processing[i].third_parties
            #msg := sprintf("Allowed processing for your purpose of use: %v", [allowed_processing])
            #msg := sprintf("Allowed processing: %v. This processing item is not allowed: %v", [allowed_processing, processing])
        }
    }

    data_items_set := {
        "device_id",
        "location",
        "email",
        "username",
        "password",
        "ip_address",
        "name",
        "address",
        "phone_number"
    }

    target_policy = {
        "advertising": [
            {"data_item": "device_id", "third_parties": ["google.com"]},
            {"data_item": "location", "third_parties": []}
        ],
        "authentication": [
            {"data_item": "email", "third_parties": []},
            {"data_item": "password", "third_parties": []},
            {"data_item": "username", "third_parties": []},
            {"data_item": "ip_address", "third_parties": []}
        ],
        "shipping": [
            {"data_item": "name", "third_parties": []},
            {"data_item": "address", "third_parties": ["canadapost-postescanada.ca"] },
            {"data_item": "email", "third_parties": ["canadapost-postescanada.ca"]},
            {"data_item": "phone_number", "third_parties": ["canadapost-postescanada.ca"]}
        ],
        "payment": [
            {"data_item": "name", "third_parties": []},
        ]
    }

