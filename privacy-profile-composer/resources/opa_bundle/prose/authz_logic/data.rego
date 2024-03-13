package prose.authz_logic

import rego.v1

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
