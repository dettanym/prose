    package prose

    import rego.v1

    test_invalid_purpose if {
        not allow with input as
        {
            "purpose_of_use": "random",
            "data_items": [],
            "external_domain": ""
        }
    }

    test_valid_purpose if {
        allow with input as
        {
            "purpose_of_use": "advertising",
            "data_items": [],
            "external_domain": ""
        }
    }

    # Test in/valid data item, in/valid third party.
    test_invalid_data_item if {
        not allow with input as
        {
            "purpose_of_use": "advertising",
            "data_items": ["random"],
            "external_domain": ""
        }
    }

    test_valid_data_item if {
        not allow with input as
        {
            "purpose_of_use": "advertising",
            "data_items": ["device_id"],
            "external_domain": ""
        }
    }

    test_invalid_third_party if {
        not allow with input as
        {
            "purpose_of_use": "advertising",
            "data_items": ["device_id"],
            "external_domain": "facebook.com"
        }
    }

    test_valid_third_party if {
        not allow with input as
        {
            "purpose_of_use": "advertising",
            "data_items": ["device_id"],
            "external_domain": "google.com"
        }
    }

    test_invalid_multiple_data_items_same_purpose if {
        not allow with input as
        {
            "purpose_of_use": "advertising",
            "data_items": ["device_id", "location"],
            "external_domain": "google.com"
        }
    }
