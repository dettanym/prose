   package authz
    import future.keywords

    test_invalid_purpose if {
        not allow with input as
        {
          "parsed_body": {
            "purpose_of_use": "random",
            "processing": []
          }
        }
    }


    test_valid_purpose if {
        allow with input as
        {
          "parsed_body": {
            "purpose_of_use": "advertising",
            "processing": []
          }
        }
    }

    # Test in/valid data item, in/valid third party.
    test_invalid_data_item if {
        not allow with input as {
          "parsed_body": {
            "purpose_of_use": "advertising",
            "processing": [
                {"data_item": "random", "third_parties": []}
            ]
          }
        }
    }

    test_valid_data_item if {
            allow with input as {
              "parsed_body": {
                "purpose_of_use": "advertising",
                "processing": [
                    {"data_item": "device_id", "third_parties": []}
                ]
              }
            }
        }

    test_invalid_third_party if {
        not allow with input as {
          "parsed_body": {
            "purpose_of_use": "advertising",
            "processing": [
                {"data_item": "device_id", "third_parties": ["facebook.com"]}
            ]
          }
        }
    }

    test_invalid_and_valid_third_parties if {
        not allow with input as
        {
          "parsed_body": {
            "purpose_of_use": "advertising",
            "processing": [
              {
                "data_item": "device_id",
                "third_parties": [
                  "facebook.com",
                  "google.com"
                ]
              }
            ]
          }
        }
    }

    test_valid_third_party if {
        allow with input as
        {
          "parsed_body": {
            "purpose_of_use": "advertising",
            "processing": [
              {
                "data_item": "device_id",
                "third_parties": [
                  "google.com"
                ]
              }
            ]
          }
        }
    }

    test_invalid_multiple_data_items_same_purpose if {
        not allow with input as
        {
          "parsed_body": {
            "purpose_of_use": "advertising",
            "processing": [
              {
                "data_item": "device_id",
                "third_parties": [
                  "google.com"
                ]
              },
              {
                "data_item": "location",
                "third_parties": [
                "google.com"
                ]
              }
            ]
          }
        }
    }

    test_valid_multiple_data_items_same_purpose if {
        allow with input as
        {
          "parsed_body": {
            "purpose_of_use": "advertising",
            "processing": [
              {
                "data_item": "device_id",
                "third_parties": [
                  "google.com"
                ]
              },
              {
                "data_item": "location",
                "third_parties": []
              }
            ]
          }
        }
    }
