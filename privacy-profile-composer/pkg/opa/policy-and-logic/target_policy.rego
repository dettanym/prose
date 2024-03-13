package prose

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
